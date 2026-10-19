#include "PlayerCore.h"

#include "AudioRenderer.h"

extern "C" {
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/channel_layout.h>
#include <libavutil/error.h>
#include <libavutil/imgutils.h>
#include <libswresample/swresample.h>
#include <libswscale/swscale.h>
}

#include <algorithm>
#include <cmath>

namespace {

std::string ffError(int code) {
    char buf[AV_ERROR_MAX_STRING_SIZE] = {0};
    av_strerror(code, buf, sizeof(buf));
    return buf;
}

} // namespace

void PacketQueue::push(AVPacket *pkt) {
    {
        std::lock_guard<std::mutex> lock(mutex);
        packets.push_back(pkt);
    }
    cv.notify_one();
}

AVPacket *PacketQueue::pop() {
    std::unique_lock<std::mutex> lock(mutex);
    cv.wait(lock, [this] { return abortFlag || !packets.empty(); });
    if (abortFlag)
        return nullptr;
    AVPacket *pkt = packets.front();
    packets.pop_front();
    return pkt;
}

void PacketQueue::reset() {
    std::deque<AVPacket *> old;
    {
        std::lock_guard<std::mutex> lock(mutex);
        old.swap(packets);
        abortFlag = false;
    }
    for (AVPacket *pkt : old) {
        if (pkt)
            av_packet_free(&pkt);
    }
}

void PacketQueue::abort() {
    {
        std::lock_guard<std::mutex> lock(mutex);
        abortFlag = true;
    }
    cv.notify_all();
}

size_t PacketQueue::size() {
    std::lock_guard<std::mutex> lock(mutex);
    return packets.size();
}

PlayerCore::PlayerCore() = default;

PlayerCore::~PlayerCore() {
    closeMedia();
}

bool PlayerCore::open(const std::string &source) {
    closeMedia();
    m_error.clear();
    if (!openDecoders(source))
        return false;
    m_opened = true;
    m_atEnd = false;
    return true;
}

bool PlayerCore::openDecoders(const std::string &source) {
    avformat_network_init();

    int ret = avformat_open_input(&m_fmt, source.c_str(), nullptr, nullptr);
    if (ret < 0) {
        m_error = "无法打开视频：" + ffError(ret);
        return false;
    }
    ret = avformat_find_stream_info(m_fmt, nullptr);
    if (ret < 0) {
        m_error = "读取视频信息失败：" + ffError(ret);
        return false;
    }

    m_vidx = av_find_best_stream(m_fmt, AVMEDIA_TYPE_VIDEO, -1, -1, nullptr, 0);
    if (m_vidx < 0) {
        m_error = "文件中没有视频流";
        return false;
    }

    AVStream *videoStream = m_fmt->streams[m_vidx];
    const AVCodec *videoCodec = avcodec_find_decoder(videoStream->codecpar->codec_id);
    if (!videoCodec) {
        m_error = "找不到视频解码器";
        return false;
    }
    m_vctx = avcodec_alloc_context3(videoCodec);
    avcodec_parameters_to_context(m_vctx, videoStream->codecpar);
    m_vctx->thread_count = 0;
    ret = avcodec_open2(m_vctx, videoCodec, nullptr);
    if (ret < 0) {
        m_error = "打开视频解码器失败：" + ffError(ret);
        return false;
    }
    m_videoWidth = m_vctx->width;
    m_videoHeight = m_vctx->height;

    m_aidx = av_find_best_stream(m_fmt, AVMEDIA_TYPE_AUDIO, -1, -1, nullptr, 0);
    if (m_aidx >= 0) {
        AVStream *audioStream = m_fmt->streams[m_aidx];
        const AVCodec *audioCodec = avcodec_find_decoder(audioStream->codecpar->codec_id);
        if (audioCodec) {
            m_actx = avcodec_alloc_context3(audioCodec);
            avcodec_parameters_to_context(m_actx, audioStream->codecpar);
            if (avcodec_open2(m_actx, audioCodec, nullptr) < 0) {
                avcodec_free_context(&m_actx);
                m_aidx = -1;
            }
        } else {
            m_aidx = -1;
        }
        if (m_aidx >= 0) {
            m_audio = std::make_unique<AudioRenderer>();
            if (!m_audio->init()) {
                m_audio.reset();
                m_aidx = -1;
            } else {
                m_audio->setVolume(m_volume / 100.0f);
            }
        }
    }

    m_startSec = m_fmt->start_time != AV_NOPTS_VALUE
                     ? m_fmt->start_time / static_cast<double>(AV_TIME_BASE)
                     : 0.0;

    if (m_fmt->duration != AV_NOPTS_VALUE)
        m_durationMs = m_fmt->duration * 1000 / AV_TIME_BASE;
    else if (videoStream->duration != AV_NOPTS_VALUE)
        m_durationMs = static_cast<int64_t>(videoStream->duration * av_q2d(videoStream->time_base) * 1000);

    m_fallbackBasePts.store(-1.0);
    m_audioClockReady.store(false);
    return true;
}

void PlayerCore::closeMedia() {
    stopThreads();
    m_audio.reset();

    if (m_vctx)
        avcodec_free_context(&m_vctx);
    if (m_actx)
        avcodec_free_context(&m_actx);
    if (m_fmt)
        avformat_close_input(&m_fmt);

    m_vidx = -1;
    m_aidx = -1;
    m_durationMs = 0;
    m_startSec = 0;
    m_opened = false;
    m_atEnd = false;
    m_playing = false;
    m_paused = false;
    m_audioClockReady = false;
    m_fallbackBasePts.store(-1.0);
    m_seekTargetSec.store(0.0);
    m_fallbackOffset = 0.0;
    m_frameSeq = 0;
    m_frameRead = 0;
    {
        std::lock_guard<std::mutex> lock(m_frameMutex);
        m_frame = Frame{};
    }
}

void PlayerCore::startThreads() {
    m_vq.reset();
    m_aq.reset();
    m_abort = false;
    m_paused = false;
    m_audioClockReady = false;
    m_threadsStarted = true;

    if (m_actx && m_audio)
        m_audioThread = std::thread(&PlayerCore::audioLoop, this);
    m_videoThread = std::thread(&PlayerCore::videoLoop, this);
    m_demuxThread = std::thread(&PlayerCore::demuxLoop, this);
}

void PlayerCore::stopThreads() {
    if (!m_threadsStarted.exchange(false)) {
        m_vq.reset();
        m_aq.reset();
        return;
    }

    m_abort = true;
    m_paused = false;
    m_vq.abort();
    m_aq.abort();

    if (m_demuxThread.joinable())
        m_demuxThread.join();
    if (m_videoThread.joinable())
        m_videoThread.join();
    if (m_audioThread.joinable())
        m_audioThread.join();

    m_abort = false;
    m_vq.reset();
    m_aq.reset();
}

void PlayerCore::demuxLoop() {
    while (!m_abort) {
        if (m_vq.size() > 150 || m_aq.size() > 200) {
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
            continue;
        }

        AVPacket *pkt = av_packet_alloc();
        int ret = av_read_frame(m_fmt, pkt);
        if (ret < 0) {
            av_packet_free(&pkt);
            break;
        }

        if (pkt->stream_index == m_vidx) {
            m_vq.push(pkt);
        } else if (pkt->stream_index == m_aidx) {
            m_aq.push(pkt);
        } else {
            av_packet_free(&pkt);
        }
    }

    if (!m_abort) {
        m_vq.push(nullptr);
        m_aq.push(nullptr);
    }
}

void PlayerCore::waitWhilePaused() const {
    while (m_paused.load() && !m_abort.load())
        std::this_thread::sleep_for(std::chrono::milliseconds(5));
}

void PlayerCore::storeFrame(const std::vector<uint8_t> &pixels, int width, int height, double pts) {
    std::lock_guard<std::mutex> lock(m_frameMutex);
    m_frame.pixels = pixels;
    m_frame.width = width;
    m_frame.height = height;
    m_frame.pts = pts;
    m_frame.seq = ++m_frameSeq;
}

bool PlayerCore::readFrame(Frame &out) {
    std::lock_guard<std::mutex> lock(m_frameMutex);
    if (m_frame.pixels.empty() || m_frame.seq == m_frameRead)
        return false;
    out = m_frame;
    m_frameRead = m_frame.seq;
    return true;
}

void PlayerCore::videoLoop() {
    AVFrame *frame = av_frame_alloc();
    SwsContext *sws = nullptr;
    std::vector<uint8_t> rgba;

    const AVRational timeBase = m_fmt->streams[m_vidx]->time_base;
    const int dstWidth = std::min(m_videoWidth, kDisplayWidth);
    const int dstHeight = std::max(1, static_cast<int>(std::lround(
                                          m_videoHeight * static_cast<double>(dstWidth) / m_videoWidth)));

    while (!m_abort) {
        AVPacket *pkt = m_vq.pop();
        if (m_abort) {
            if (pkt)
                av_packet_free(&pkt);
            break;
        }
        const bool eof = (pkt == nullptr);

        int ret = pkt ? avcodec_send_packet(m_vctx, pkt) : avcodec_send_packet(m_vctx, nullptr);
        if (pkt)
            av_packet_free(&pkt);
        if (ret < 0 && ret != AVERROR_EOF)
            continue;

        while (!m_abort) {
            ret = avcodec_receive_frame(m_vctx, frame);
            if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF)
                break;
            if (ret < 0)
                break;

            int64_t ts = frame->pts;
            if (ts == AV_NOPTS_VALUE)
                ts = frame->best_effort_timestamp;
            const double ptsSec = (ts == AV_NOPTS_VALUE ? 0.0 : ts * av_q2d(timeBase)) - m_startSec;

            if (ptsSec + 0.02 < m_seekTargetSec.load()) {
                av_frame_unref(frame);
                continue;
            }

            if (m_fallbackBasePts.load() < 0.0)
                m_fallbackBasePts.store(ptsSec);

            waitWhilePaused();
            if (m_abort) {
                av_frame_unref(frame);
                break;
            }

            while (!m_abort && !m_paused) {
                const double delay = ptsSec - mediaClock();
                if (delay <= 0.004)
                    break;
                std::this_thread::sleep_for(std::chrono::milliseconds(delay > 0.04 ? 4 : 1));
            }
            if (m_abort) {
                av_frame_unref(frame);
                break;
            }

            if (ptsSec < mediaClock() - 0.25) {
                av_frame_unref(frame);
                continue;
            }

            if (!sws) {
                sws = sws_getContext(m_videoWidth, m_videoHeight, m_vctx->pix_fmt,
                                     dstWidth, dstHeight, AV_PIX_FMT_RGBA,
                                     SWS_BILINEAR, nullptr, nullptr, nullptr);
                if (!sws) {
                    av_frame_unref(frame);
                    break;
                }
            }

            rgba.resize(static_cast<size_t>(dstWidth) * dstHeight * 4);
            uint8_t *dstData[4] = {rgba.data(), nullptr, nullptr, nullptr};
            int dstLines[4] = {dstWidth * 4, 0, 0, 0};
            sws_scale(sws, frame->data, frame->linesize, 0, m_videoHeight, dstData, dstLines);
            storeFrame(rgba, dstWidth, dstHeight, ptsSec);
            av_frame_unref(frame);
        }

        if (eof)
            break;
    }

    av_frame_free(&frame);
    if (sws)
        sws_freeContext(sws);

    if (!m_abort) {
        m_atEnd = true;
        m_playing = false;
        if (m_audio)
            m_audio->stop();
    }
}

void PlayerCore::audioLoop() {
    AVFrame *frame = av_frame_alloc();
    SwrContext *swr = nullptr;
    std::vector<int16_t> outSamples;

    const AVRational timeBase = m_fmt->streams[m_aidx]->time_base;

    while (!m_abort) {
        AVPacket *pkt = m_aq.pop();
        if (m_abort) {
            if (pkt)
                av_packet_free(&pkt);
            break;
        }
        const bool eof = (pkt == nullptr);

        int ret = pkt ? avcodec_send_packet(m_actx, pkt) : avcodec_send_packet(m_actx, nullptr);
        if (pkt)
            av_packet_free(&pkt);
        if (ret < 0 && ret != AVERROR_EOF)
            continue;

        while (!m_abort) {
            ret = avcodec_receive_frame(m_actx, frame);
            if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF)
                break;
            if (ret < 0)
                break;

            if (!swr) {
                AVChannelLayout outLayout = AV_CHANNEL_LAYOUT_STEREO;
                if (swr_alloc_set_opts2(&swr, &outLayout, AV_SAMPLE_FMT_S16, 48000,
                                        &m_actx->ch_layout, m_actx->sample_fmt,
                                        m_actx->sample_rate, 0, nullptr) < 0 ||
                    swr_init(swr) < 0) {
                    av_frame_unref(frame);
                    break;
                }
            }

            int64_t apts = frame->pts;
            if (apts == AV_NOPTS_VALUE)
                apts = frame->best_effort_timestamp;
            const double aptsSec = (apts == AV_NOPTS_VALUE ? 0.0 : apts * av_q2d(timeBase)) - m_startSec;
            if (aptsSec + 0.02 < m_seekTargetSec.load()) {
                av_frame_unref(frame);
                continue;
            }

            const int outCount = std::max(1, swr_get_out_samples(swr, frame->nb_samples));
            outSamples.resize(static_cast<size_t>(outCount) * 2);
            uint8_t *outPtr = reinterpret_cast<uint8_t *>(outSamples.data());
            const int converted = swr_convert(swr, &outPtr, outCount,
                                              const_cast<const uint8_t **>(frame->extended_data),
                                              frame->nb_samples);

            if (converted > 0) {
                if (!m_audioClockReady.load()) {
                    m_audioBasePts.store(aptsSec);
                    m_audioClockReady.store(true);
                }
                m_audio->write(outSamples.data(), static_cast<size_t>(converted));
            }
            av_frame_unref(frame);

            while (!m_abort && m_audio && m_audio->bufferedFrames() > 48000 * 0.8)
                std::this_thread::sleep_for(std::chrono::milliseconds(5));
        }

        if (eof)
            break;
    }

    if (swr)
        swr_free(&swr);
    av_frame_free(&frame);
}

double PlayerCore::mediaClock() const {
    if (m_audio && m_audioClockReady.load()) {
        return m_audioBasePts.load() + static_cast<double>(m_audio->framesPlayed()) / 48000.0;
    }

    const double base = m_fallbackBasePts.load();
    if (base < 0.0)
        return 0.0;

    std::lock_guard<std::mutex> lock(m_clockMutex);
    double elapsed = m_fallbackOffset;
    if (m_playing.load())
        elapsed += std::chrono::duration<double>(std::chrono::steady_clock::now() - m_playStartedAt).count();
    return base + elapsed;
}

void PlayerCore::play() {
    if (!m_opened)
        return;

    if (m_atEnd)
        seek(0);

    if (!m_threadsStarted)
        startThreads();

    m_paused = false;
    {
        std::lock_guard<std::mutex> lock(m_clockMutex);
        m_playStartedAt = std::chrono::steady_clock::now();
    }
    if (m_audio)
        m_audio->start();
    m_playing = true;
    m_atEnd = false;
}

void PlayerCore::pause() {
    m_paused = true;
    if (m_audio)
        m_audio->stop();
    if (m_playing.exchange(false)) {
        std::lock_guard<std::mutex> lock(m_clockMutex);
        m_fallbackOffset += std::chrono::duration<double>(
                                std::chrono::steady_clock::now() - m_playStartedAt)
                                .count();
    }
}

void PlayerCore::seek(int64_t ms) {
    if (!m_opened)
        return;

    ms = std::clamp<int64_t>(ms, 0, std::max<int64_t>(0, m_durationMs));
    const bool wasPlaying = m_playing.load();

    m_paused = true;
    if (m_audio)
        m_audio->stop();
    stopThreads();

    const int64_t target = static_cast<int64_t>((m_startSec + ms / 1000.0) * AV_TIME_BASE);
    if (av_seek_frame(m_fmt, -1, target, AVSEEK_FLAG_BACKWARD) < 0)
        avformat_seek_file(m_fmt, -1, INT64_MIN, target, INT64_MAX, 0);

    avcodec_flush_buffers(m_vctx);
    if (m_actx)
        avcodec_flush_buffers(m_actx);

    if (m_audio) {
        m_audio->clear();
        m_audio->resetClock();
    }
    m_audioClockReady = false;
    m_audioBasePts = 0.0;
    m_fallbackBasePts = ms / 1000.0;
    m_seekTargetSec = ms / 1000.0;
    {
        std::lock_guard<std::mutex> lock(m_clockMutex);
        m_fallbackOffset = 0.0;
        m_playStartedAt = std::chrono::steady_clock::now();
    }
    {
        std::lock_guard<std::mutex> lock(m_frameMutex);
        m_frame = Frame{};
        m_frameSeq = 0;
        m_frameRead = 0;
    }
    m_atEnd = false;

    if (wasPlaying) {
        m_paused = false;
        m_playing = true;
        startThreads();
        if (m_audio)
            m_audio->start();
    } else {
        m_playing = false;
    }
}

int64_t PlayerCore::positionMs() const {
    if (!m_opened)
        return 0;
    if (m_atEnd.load())
        return m_durationMs;

    const double seconds = mediaClock();
    const int64_t ms = static_cast<int64_t>(std::max(0.0, seconds) * 1000.0);
    return std::clamp<int64_t>(ms, 0, m_durationMs);
}

void PlayerCore::setVolume(int volume) {
    m_volume = std::clamp(volume, 0, 100);
    if (m_audio)
        m_audio->setVolume(m_volume / 100.0f);
}

std::string PlayerCore::lastError() const {
    return m_error;
}
