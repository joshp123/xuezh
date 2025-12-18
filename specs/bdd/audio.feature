Feature: Audio processing pipeline

  Scenario: Process a voice note for pronunciation practice (audio in, audio out)
    Given a clean workspace

    # Input: Telegram-style voice note (ogg/opus)
    When the client runs "xuezh audio process-voice --in tests/fixtures/audio/voice_min.ogg --ref-text '你好' --backend local --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

    # The schema for audio.process-voice requires the backend to output these artifacts:
    # - normalized_input (wav)
    # - transcript (json)
    # - assessment (json)
    # - feedback_voice_note (ogg/opus)


Scenario: Convert audio file formats (mechanical)
  Given a clean workspace
  When the client runs "xuezh audio convert --in tests/fixtures/audio/sine_440hz.wav --out {workspace}/artifacts/converted.ogg --format ogg --backend ffmpeg --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: TTS produces a voice note artifact
  Given a clean workspace
  When the client runs "xuezh audio tts --text '你好' --voice XiaoxiaoNeural --out {workspace}/artifacts/tts.ogg --backend edge-tts --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: STT returns a transcript artifact
  Given a clean workspace
  When the client runs "xuezh audio stt --in tests/fixtures/audio/sine_440hz.wav --backend whisper --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: Pronunciation assessment returns an assessment artifact
  Given a clean workspace
  When the client runs "xuezh audio assess --ref-text '你好' --in tests/fixtures/audio/voice_min.ogg --backend local --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields
