Feature: Spaced repetition review primitives

  Scenario: Review due items and record a grade
    Given a clean workspace
    When the client runs "xuezh db init --json"
    Then the engine returns a valid JSON envelope

    # This scenario becomes strict (non-xfail) once review is implemented.
    When the client runs "xuezh review start --limit 10 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

    When the client runs "xuezh review grade --item w_aaaaaaaaaaaa --grade 4 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields


Scenario: Bury an item
  Given a clean workspace
  When the client runs "xuezh db init --json"
  Then the engine returns a valid JSON envelope

  When the client runs "xuezh review bury --item w_aaaaaaaaaaaa --reason hard --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields

Scenario: Preview SRS load
  Given a clean workspace
  When the client runs "xuezh db init --json"
  Then the engine returns a valid JSON envelope

  When the client runs "xuezh srs preview --days 14 --json"
  Then the engine returns an OK envelope
  And the output matches the command-specific JSON schema
  And the engine does not return recommendation fields
