Feature: Event logging (exposures)

  Scenario: Log an exposure event and list it back
    Given a clean workspace
    When the client runs "xuezh db init --json"
    Then the engine returns a valid JSON envelope

    When the client runs "xuezh event log --type exposure --modality reading --items w_aaaaaaaaaaaa --context ct_deadbeefcafe --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

    When the client runs "xuezh event list --since 7d --limit 200 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
