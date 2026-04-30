Feature: HelloChinese cram review
  The engine imports HelloChinese rows and exposes a fast deterministic cram loop.

  Scenario: Import a HelloChinese fixture without audio
    When I run `xuezh hellochinese import --path internal/xuezh/hellochinese/testdata/min.jsonl --audio none --json`
    Then the JSON envelope is ok
    And the command is `hellochinese.import`
    And `data.rows_seen` is `3`

  Scenario: Fetch the next cram card
    Given the HelloChinese fixture is imported
    When I run `xuezh cram next --limit 1 --json`
    Then the JSON envelope is ok
    And the command is `cram.next`
    And the first card has sentence `你是龙大。`
    And the output does not contain `recommended_next`

  Scenario: Grade a cram card
    Given the HelloChinese fixture is imported
    And I have fetched the first cram card
    When I run `xuezh cram grade --item <item_id> --grade good --json`
    Then the JSON envelope is ok
    And the command is `cram.grade`
    And `data.grade` is `good`

  Scenario: Backfill sentence audio for imported rows
    Given the HelloChinese fixture is imported
    When I run `xuezh hellochinese audio-backfill --limit 3 --concurrency 4 --json`
    Then the JSON envelope is ok
    And the command is `hellochinese.audio-backfill`
