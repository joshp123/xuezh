Feature: Content cache

  Scenario: Put and get a cached story
    Given a clean workspace
    When the client runs "xuezh db init --json"
    Then the engine returns a valid JSON envelope

    When the client runs "xuezh content cache put --type story --key story_min --in tests/fixtures/content/story_min.txt --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

    When the client runs "xuezh content cache get --type story --key story_min --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
