#pragma once

#include <atomic>
#include <cstdint>
#include <deque>
#include <mutex>

struct ma_device;

class AudioRenderer {
public:
    AudioRenderer();
    ~AudioRenderer();

    bool init(int sampleRate = 48000, int channels = 2);
    void start();
    void stop();
    void clear();
    void resetClock();
    void setVolume(float volume);
    size_t bufferedFrames();
    uint64_t framesPlayed() const { return m_framesPlayed.load(); }

    void write(const int16_t *samples, size_t frames);

private:
    static void dataCallback(ma_device *device, void *output, const void *input, uint32_t frameCount);

    ma_device *m_device = nullptr;
    std::mutex m_mutex;
    std::deque<int16_t> m_buffer;
    std::atomic<uint64_t> m_framesPlayed{0};
    std::atomic<bool> m_started{false};
    int m_sampleRate = 48000;
    int m_channels = 2;
};
