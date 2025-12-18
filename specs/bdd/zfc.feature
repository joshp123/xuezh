Feature: ZFC boundary

  Scenario: Engine must not provide recommendations
    Given a clean workspace
    When the client runs "xuezh snapshot --window 30d --due-limit 10 --evidence-limit 50 --max-bytes 200000 --json"
    Then the engine returns a valid JSON envelope
    And the engine does not return recommendation fields
