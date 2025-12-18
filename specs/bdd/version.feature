Feature: Version command

  Scenario: Print version in JSON envelope
    Given a clean workspace
    When the client runs "chlearn version --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
