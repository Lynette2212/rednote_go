#include <napi.h>

#include "PlayerCore.h"

class PlayerWrap : public Napi::ObjectWrap<PlayerWrap> {
public:
    static Napi::Object Init(Napi::Env env, Napi::Object exports) {
        Napi::Function constructor = DefineClass(env, "Player", {
            InstanceMethod("open", &PlayerWrap::Open),
            InstanceMethod("play", &PlayerWrap::Play),
            InstanceMethod("pause", &PlayerWrap::Pause),
            InstanceMethod("seek", &PlayerWrap::Seek),
            InstanceMethod("setVolume", &PlayerWrap::SetVolume),
            InstanceMethod("getFrame", &PlayerWrap::GetFrame),
            InstanceMethod("getState", &PlayerWrap::GetState),
        });
        exports.Set("Player", constructor);
        static Napi::FunctionReference playerConstructor;
        playerConstructor = Napi::Persistent(constructor);
        exports.Set("createPlayer", Napi::Function::New(env, [](const Napi::CallbackInfo &info) -> Napi::Value {
            return playerConstructor.New({});
        }));
        return exports;
    }

    explicit PlayerWrap(const Napi::CallbackInfo &info)
        : Napi::ObjectWrap<PlayerWrap>(info), m_core(new PlayerCore()) {
    }

    ~PlayerWrap() override {
        delete m_core;
    }

private:
    Napi::Value Open(const Napi::CallbackInfo &info) {
        Napi::Env env = info.Env();
        if (info.Length() < 1 || !info[0].IsString())
            Napi::TypeError::New(env, "path required").ThrowAsJavaScriptException();
        const std::string path = info[0].As<Napi::String>().Utf8Value();
        return Napi::Boolean::New(env, m_core->open(path));
    }

    Napi::Value Play(const Napi::CallbackInfo &info) {
        m_core->play();
        return info.Env().Undefined();
    }

    Napi::Value Pause(const Napi::CallbackInfo &info) {
        m_core->pause();
        return info.Env().Undefined();
    }

    Napi::Value Seek(const Napi::CallbackInfo &info) {
        if (info.Length() >= 1 && info[0].IsNumber())
            m_core->seek(info[0].As<Napi::Number>().Int64Value());
        return info.Env().Undefined();
    }

    Napi::Value SetVolume(const Napi::CallbackInfo &info) {
        if (info.Length() >= 1 && info[0].IsNumber())
            m_core->setVolume(info[0].As<Napi::Number>().Int32Value());
        return info.Env().Undefined();
    }

    Napi::Value GetFrame(const Napi::CallbackInfo &info) {
        Napi::Env env = info.Env();
        PlayerCore::Frame frame;
        if (!m_core->readFrame(frame))
            return env.Null();

        Napi::Object result = Napi::Object::New(env);
        result.Set("buffer", Napi::Buffer<uint8_t>::Copy(env, frame.pixels.data(), frame.pixels.size()));
        result.Set("width", Napi::Number::New(env, frame.width));
        result.Set("height", Napi::Number::New(env, frame.height));
        result.Set("pts", Napi::Number::New(env, frame.pts));
        result.Set("seq", Napi::Number::New(env, static_cast<double>(frame.seq)));
        return result;
    }

    Napi::Value GetState(const Napi::CallbackInfo &info) {
        Napi::Env env = info.Env();
        Napi::Object state = Napi::Object::New(env);
        state.Set("playing", Napi::Boolean::New(env, m_core->isPlaying()));
        state.Set("ended", Napi::Boolean::New(env, m_core->isEnded()));
        state.Set("positionMs", Napi::Number::New(env, static_cast<double>(m_core->positionMs())));
        state.Set("durationMs", Napi::Number::New(env, static_cast<double>(m_core->durationMs())));
        state.Set("error", Napi::String::New(env, m_core->lastError()));
        return state;
    }

    PlayerCore *m_core;
};

Napi::Object InitAll(Napi::Env env, Napi::Object exports) {
    return PlayerWrap::Init(env, exports);
}

NODE_API_MODULE(rednote_player, InitAll)
