Feature: Doctor command (diagnostics)

  Scenario: Run diagnostics in JSON envelope
    Given a clean workspace
    When the client runs "xuezh doctor --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
