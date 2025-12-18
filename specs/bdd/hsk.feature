Feature: HSK audit reporting

  Scenario: Request an HSK compliance report
    Given a clean workspace
    When the client runs "chlearn db init --json"
    Then the engine returns a valid JSON envelope

    When the client runs "chlearn dataset import --type hsk_vocab --path tests/fixtures/datasets/hsk_vocab_min.csv --json"
    Then the engine returns an OK envelope
    And the engine does not return recommendation fields

    When the client runs "chlearn report hsk --level 3 --window 30d --max-items 200 --max-bytes 200000 --json"
    Then the engine returns an OK envelope
    And the output matches the command-specific JSON schema
    And the engine does not return recommendation fields

        # report.hsk schema requires coverage.vocab and coverage.grammar (chars optional via --include-chars).
