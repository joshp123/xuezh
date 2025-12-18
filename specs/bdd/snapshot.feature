Feature: Snapshot command

  Scenario: Get a bounded snapshot context pack
    Given a clean workspace
    When the client runs "xuezh snapshot --window 30d --due-limit 20 --evidence-limit 50 --max-bytes 200000 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
