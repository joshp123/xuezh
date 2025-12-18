Feature: Audio processing pipeline

  Scenario: Process a voice note for pronunciation practice (audio in, audio out)
    Given a clean workspace

    # Input: Telegram-style voice note (ogg/opus)
    When the client runs "chlearn audio process-voice --in tests/fixtures/audio/voice_min.ogg --ref-text '你好' --mode local --json"
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
  When the client runs "chlearn audio convert --in tests/fixtures/audio/sine_440hz.wav --out {workspace}/artifacts/converted.ogg --format ogg --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: TTS produces a voice note artifact
  Given a clean workspace
  When the client runs "chlearn audio tts --text '你好' --voice XiaoxiaoNeural --out {workspace}/artifacts/tts.ogg --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: STT returns a transcript artifact
  Given a clean workspace
  When the client runs "chlearn audio stt --in tests/fixtures/audio/sine_440hz.wav --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: Pronunciation assessment returns an assessment artifact
  Given a clean workspace
  When the client runs "chlearn audio assess --ref-text '你好' --in tests/fixtures/audio/voice_min.ogg --mode local --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields
