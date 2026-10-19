#include "AudioRenderer.h"

#ifndef NOMINMAX
#define NOMINMAX
#endif
#define MINIAUDIO_IMPLEMENTATION
#include "miniaudio.h"

#include <algorithm>

AudioRenderer::AudioRenderer() = default;

AudioRenderer::~AudioRenderer() {
    if (m_device) {
        ma_device_stop(m_device);
        ma_device_uninit(m_device);
        delete m_device;
        m_device = nullptr;
    }
}

bool AudioRenderer::init(int sampleRate, int channels) {
    if (m_device)
        return true;

    m_sampleRate = sampleRate;
    m_channels = channels;

    m_device = new ma_device;
    ma_device_config config = ma_device_config_init(ma_device_type_playback);
    config.playback.format = ma_format_s16;
    config.playback.channels = static_cast<ma_uint32>(channels);
    config.sampleRate = static_cast<ma_uint32>(sampleRate);
    config.dataCallback = &AudioRenderer::dataCallback;
    config.pUserData = this;

    if (ma_device_init(nullptr, &config, m_device) != MA_SUCCESS) {
        delete m_device;
        m_device = nullptr;
        return false;
    }
    return true;
}

void AudioRenderer::start() {
    if (!m_device || m_started.load())
        return;
    if (ma_device_start(m_device) == MA_SUCCESS)
        m_started.store(true);
}

void AudioRenderer::stop() {
    if (!m_device || !m_started.load())
        return;
    ma_device_stop(m_device);
    m_started.store(false);
}

void AudioRenderer::clear() {
    std::lock_guard<std::mutex> lock(m_mutex);
    m_buffer.clear();
}

void AudioRenderer::resetClock() {
    m_framesPlayed.store(0);
}

void AudioRenderer::setVolume(float volume) {
    if (!m_device)
        return;
    ma_device_set_master_volume(m_device, std::clamp(volume, 0.0f, 1.0f));
}

size_t AudioRenderer::bufferedFrames() {
    std::lock_guard<std::mutex> lock(m_mutex);
    return m_buffer.size() / static_cast<size_t>(m_channels);
}

void AudioRenderer::write(const int16_t *samples, size_t frames) {
    std::lock_guard<std::mutex> lock(m_mutex);
    const size_t maxSamples = static_cast<size_t>(m_sampleRate) * m_channels * 2;
    if (m_buffer.size() >= maxSamples)
        return;
    const size_t count = std::min(frames * static_cast<size_t>(m_channels), maxSamples - m_buffer.size());
    m_buffer.insert(m_buffer.end(), samples, samples + count);
}

void AudioRenderer::dataCallback(ma_device *device, void *output, const void *, uint32_t frameCount) {
    auto *self = static_cast<AudioRenderer *>(device->pUserData);
    auto *dst = static_cast<int16_t *>(output);
    const size_t channels = static_cast<size_t>(self->m_channels);
    size_t framesRead = 0;

    {
        std::lock_guard<std::mutex> lock(self->m_mutex);
        for (uint32_t i = 0; i < frameCount; ++i) {
            if (self->m_buffer.size() >= channels) {
                for (size_t ch = 0; ch < channels; ++ch) {
                    dst[i * channels + ch] = self->m_buffer.front();
                    self->m_buffer.pop_front();
                }
                ++framesRead;
            } else {
                for (size_t ch = 0; ch < channels; ++ch)
                    dst[i * channels + ch] = 0;
            }
        }
    }
    self->m_framesPlayed.fetch_add(framesRead);
}
