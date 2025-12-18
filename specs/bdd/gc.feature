Feature: Artifact retention and garbage collection

  Scenario: GC dry-run returns candidate list without deleting
    Given a clean workspace
    When the client runs "chlearn gc --dry-run --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields
