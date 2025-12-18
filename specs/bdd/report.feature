Feature: Reports (mastery and due)

  Scenario: Mastery report for words
    Given a clean workspace
    When the client runs "chlearn db init --json"
    Then the engine returns a valid JSON envelope

    When the client runs "chlearn report mastery --item-type word --window 90d --max-items 50 --max-bytes 200000 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

  Scenario: Due report
    Given a clean workspace
    When the client runs "chlearn db init --json"
    Then the engine returns a valid JSON envelope

    When the client runs "chlearn report due --limit 50 --max-bytes 200000 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
