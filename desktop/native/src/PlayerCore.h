#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <deque>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

struct AVFormatContext;
struct AVCodecContext;
struct AVPacket;

class AudioRenderer;

struct PacketQueue {
    std::deque<AVPacket *> packets;
    std::mutex mutex;
    std::condition_variable cv;
    bool abortFlag = false;

    void push(AVPacket *pkt);
    AVPacket *pop();
    void reset();
    void abort();
    size_t size();
};

class PlayerCore {
public:
    struct Frame {
        std::vector<uint8_t> pixels;
        int width = 0;
        int height = 0;
        double pts = 0.0;
        uint64_t seq = 0;
    };

    PlayerCore();
    ~PlayerCore();

    bool open(const std::string &source);
    void play();
    void pause();
    void seek(int64_t ms);
    void setVolume(int volume);

    bool isPlaying() const { return m_playing.load(); }
    bool isEnded() const { return m_atEnd.load(); }
    int64_t durationMs() const { return m_durationMs; }
    int64_t positionMs() const;
    std::string lastError() const;

    // 返回自上次调用以来的最新一帧（RGBA）；没有新帧返回 false
    bool readFrame(Frame &out);

private:
    void closeMedia();
    bool openDecoders(const std::string &source);
    void startThreads();
    void stopThreads();
    void demuxLoop();
    void videoLoop();
    void audioLoop();
    double mediaClock() const;
    void waitWhilePaused() const;
    void storeFrame(const std::vector<uint8_t> &pixels, int width, int height, double pts);

    AVFormatContext *m_fmt = nullptr;
    AVCodecContext *m_vctx = nullptr;
    AVCodecContext *m_actx = nullptr;
    int m_vidx = -1;
    int m_aidx = -1;
    double m_startSec = 0.0;
    int64_t m_durationMs = 0;
    int m_videoWidth = 0;
    int m_videoHeight = 0;

    std::unique_ptr<AudioRenderer> m_audio;

    PacketQueue m_vq;
    PacketQueue m_aq;
    std::thread m_demuxThread;
    std::thread m_videoThread;
    std::thread m_audioThread;

    std::atomic<bool> m_abort{false};
    std::atomic<bool> m_paused{false};
    std::atomic<bool> m_threadsStarted{false};
    std::atomic<bool> m_playing{false};
    std::atomic<bool> m_atEnd{false};
    std::atomic<bool> m_audioClockReady{false};
    std::atomic<double> m_audioBasePts{0.0};
    std::atomic<double> m_fallbackBasePts{-1.0};
    std::atomic<double> m_seekTargetSec{0.0};

    mutable std::mutex m_clockMutex;
    double m_fallbackOffset = 0.0;
    std::chrono::steady_clock::time_point m_playStartedAt;

    mutable std::mutex m_frameMutex;
    Frame m_frame;
    uint64_t m_frameSeq = 0;
    uint64_t m_frameRead = 0;

    std::string m_error;
    int m_volume = 70;
    bool m_opened = false;

    static constexpr int kDisplayWidth = 540;
};
