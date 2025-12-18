# NOT A SPEC (reference/background)
#
# This file is preserved as background context and historical notes.
# Authoritative requirements live in:
# - `docs/cli-contract.md`
# - `specs/cli/contract.json`
# - `schemas/`
# - `specs/bdd/`

# Comprehensive Language Learning System Design Request

## Project Overview
Design and implement a complete, production-ready language learning system for Chinese (Mandarin) that revolutionizes how adults learn languages by mimicking child language acquisition patterns. This system should be implemented as a skill/tool for an AI assistant (Claude Opus 4.0) running through a Telegram bot interface.

## Detailed Context

### User Profile
- Adult learner (Jos) with intermediate Chinese knowledge
- Knows basic vocabulary (~500-1000 words)
- Can form simple sentences
- Struggles with tones and pronunciation
- Has family of 7 (uses 七口人 correctly - advanced!)
- Located in Europe (likely Netherlands/Belgium based on timezone)
- Wants flexible, on-demand learning
- Prefers natural acquisition over rote memorization

### Current Technical Environment
- **Primary Interface**: Telegram bot (@jospalmbot)
- **AI Backend**: Claude Opus 4.0 via Pi-coding-agent
- **Gateway**: Clawdis (custom Telegram/WhatsApp gateway)
- **Operating System**: macOS on M4 MacBook Pro
- **Available Tools**:
  - edge-tts: Microsoft's neural TTS with Chinese voices (XiaoxiaoNeural, YunxiNeural, YunyangNeural)
  - whisper: OpenAI's STT for transcription
  - ffmpeg: Audio/video processing
  - yt-dlp: YouTube content downloading
  - devenv.nix: For installing additional tools
  - Python 3.11 environment
  - Node.js environment
  - Full filesystem access
  - Image generation/processing capabilities

### Technical Limitations & Gaps
1. **No pronunciation analysis**: Whisper only transcribes, doesn't analyze pronunciation quality
2. **No spaced repetition system**: Need to build from scratch
3. **No persistent database**: Currently using JSON/markdown files
4. **No real-time feedback**: Can't analyze speech while user is speaking
5. **Limited scheduling**: Can't send proactive messages at specific times
6. **No gamification framework**: Need to design engagement mechanics

## Comprehensive Requirements

### Core Learning Philosophy
The system must embody these principles:
1. **Natural Acquisition**: Learn like children do - through exposure, pattern recognition, and contextual understanding
2. **Comprehensible Input**: Always provide material slightly above current level (i+1)
3. **Emotional Engagement**: Make content personally relevant and emotionally resonant
4. **Active Production**: Balance input with output opportunities
5. **Error Tolerance**: Mistakes are learning opportunities, not failures
6. **Contextual Learning**: Grammar emerges from patterns, not explicit rules
7. **Multi-sensory**: Engage visual, auditory, and kinesthetic learning

### Vocabulary & Character Management

#### Data Structure Requirements
```python
class Character:
    character: str
    pinyin: List[str]  # Multiple readings
    meanings: List[str]
    radical: str
    stroke_count: int
    frequency_rank: int
    
class Word:
    characters: str
    pinyin: str
    meanings: List[str]
    part_of_speech: List[str]
    example_sentences: List[str]
    audio_file: str
    
class UserKnowledge:
    word_id: str
    character_id: str
    first_seen: datetime
    last_seen: datetime
    times_seen: int
    times_correct: int
    familiarity_score: float  # 0.0 to 1.0
    contexts_seen: List[str]
    pronunciation_scores: List[float]
```

#### Tracking Requirements
- Separate tracking for word recognition vs character recognition
- Track passive knowledge (understanding) vs active knowledge (production)
- Context-based familiarity (know word in one context but not another)
- Pronunciation accuracy tracking per word/tone
- Common mistake patterns

### Learning Activities Design

#### 1. Graded Readers
- Auto-generate stories using known vocabulary + 5-10% new words
- Genres: Daily life, fantasy, humor, personal interests
- Length: Start with 50-100 characters, scale up
- Include audio narration with adjustable speed
- Highlight new words/characters
- Inline definitions on tap/hover

#### 2. Conversational Practice
- Scenario-based dialogues (restaurant, shopping, travel)
- AI plays different characters with personalities
- Speech recognition with tone checking
- Natural error correction through reformulation
- Branching conversations based on user responses

#### 3. Listening Comprehension
- Podcast-style content at various speeds
- Real-world audio (news, YouTube clips)
- Interactive transcripts
- Comprehension questions that test understanding, not memory
- Shadowing exercises with scoring

#### 4. Speaking Practice
- Pronunciation drills with visual tone guides
- Minimal pair discrimination (distinguish similar sounds)
- Sentence pattern drills with substitution
- Free speaking prompts with AI feedback
- Tone sandhi practice (like 不 becoming bú)

#### 5. Writing Practice
- Character stroke order animation
- Composition practice with prompts
- Text message simulation
- Diary/journal entries
- Character decomposition exercises

#### 6. Grammar Through Patterns
- Present patterns in context, not rules
- Sentence transformation exercises
- Fill-in-the-blank with multiple correct answers
- Pattern recognition games
- Contrastive examples (correct vs incorrect usage)

### Progress & Motivation System

#### Metrics to Track
- Daily active time
- New words/characters learned
- Retention rates at 1 day, 1 week, 1 month
- Pronunciation accuracy trends
- Reading speed progression
- Listening comprehension scores
- Speaking fluency (words per minute, pause frequency)

#### Gamification Elements
- XP system for all activities
- Streak tracking (but with vacation mode)
- Achievement badges for milestones
- Character collection (unlock rare characters)
- Story progression unlocked by progress
- Leaderboard with past self (not other users)
- Virtual pet that grows with consistent practice

### Spaced Repetition Algorithm

Implement a modified SM2 algorithm with these adjustments:
- Consider context (word easier in familiar context)
- Adjust for interference (similar words need more spacing)
- Factor in pronunciation difficulty separately
- Allow manual adjustment of intervals
- Preview upcoming reviews
- Batch reviews by theme/context

### Technical Architecture

#### Database Design
Use SQLite with these tables:
- users
- characters
- words  
- user_knowledge
- learning_sessions
- pronunciation_attempts
- generated_content
- error_patterns

#### Audio Processing Pipeline
1. User sends voice message
2. Whisper transcribes to text
3. Custom pronunciation analyzer compares to reference
4. Generate feedback audio with correct pronunciation
5. Store attempt for progress tracking

#### Content Generation System
- Use Claude to generate level-appropriate content
- Template system for different exercise types
- Caching to avoid regenerating common content
- Quality scoring to filter generated content

#### The Missing Pronunciation Tool
Research indicates these options:
1. **Gentle**: Forced alignment tool for pronunciation assessment
2. **SpeechRater**: Academic tool for pronunciation scoring
3. **Praat**: Phonetic analysis software with Python bindings
4. **Azure Speech Services**: Has pronunciation assessment API
5. **Google Cloud Speech**: Includes confidence scores

### Implementation Roadmap

#### Phase 1: Foundation (Week 1)
1. Set up SQLite database
2. Create basic data models
3. Implement vocabulary tracking
4. Build simple spaced repetition
5. Create first learning activities

#### Phase 2: Core Features (Week 2-3)
1. Integrate pronunciation analysis tool
2. Build content generation system
3. Implement progress tracking
4. Create gamification elements
5. Develop first 10 lesson templates

#### Phase 3: Polish (Week 4)
1. Refine UI/UX in Telegram
2. Add achievement system
3. Implement analytics dashboard
4. Create onboarding flow
5. Beta test with user

### Skill.md Structure

```markdown
# Chinese Learning System Skill

## Overview
Comprehensive Chinese learning system with natural acquisition focus.

## Commands
- `/learn` - Start learning session
- `/review` - Spaced repetition review
- `/stats` - View progress
- `/speak <text>` - Practice pronunciation
- `/story` - Generate graded reader
- `/chat` - Conversational practice

## Workflows

### Daily Learning Flow
1. Check for due reviews
2. Present 1-2 new characters/words
3. Generate contextual exercises
4. Practice pronunciation
5. Read/listen to graded content
6. Free conversation practice
7. Log progress and plan next session

### Pronunciation Practice Flow
1. Present target phrase
2. Play reference audio
3. Record user attempt
4. Analyze pronunciation
5. Provide specific feedback
6. Repeat until satisfactory

## Data Management
- Store all data in `~/.clawdis/workspace/chinese-learning/`
- Daily backups to `chinese-learning-backup/`
- Export progress reports weekly
```

## Specific Questions to Address

### What pronunciation analysis tool should we use?
Recommend **Azure Speech Services** for these reasons:
- Built-in pronunciation assessment
- Supports Mandarin with tone analysis
- Real-time feedback capability
- Good API with Python SDK
- Reasonable pricing for personal use

Alternative: Build custom solution with Gentle + Praat for more control.

### How to implement spaced repetition elegantly?
Use a modified Leitner system:
- 5 boxes representing familiarity levels
- Move cards between boxes based on performance
- Review schedule: Box 1 (daily), Box 2 (every 3 days), Box 3 (weekly), Box 4 (biweekly), Box 5 (monthly)
- Cards drop one box on error
- Context-aware: same word might be in different boxes for different contexts

### Best database solution?
SQLite is ideal because:
- No server required
- Single file storage
- Full SQL support
- Python/Node.js bindings
- Easy backup
- Can migrate to PostgreSQL later if needed

### How to auto-generate graded content?
1. Analyze user's known vocabulary
2. Select theme/scenario
3. Use Claude to generate story with constraints:
   - 90% known words
   - 10% new words (from frequency list)
   - Repetition of new words in different contexts
   - Natural language patterns
4. Post-process to add pinyin/definitions
5. Generate audio with edge-tts

### How to make it genuinely fun?
1. **Personal relevance**: Generate content about user's interests
2. **Humor**: Include jokes, puns, funny scenarios
3. **Mystery**: Serial stories that unfold with progress
4. **Social**: Share achievements, compete with past self
5. **Variety**: Different activity types to prevent boredom
6. **Surprise**: Random rewards, easter eggs
7. **Agency**: Let user choose learning path

## Immediate Next Steps

1. Install Azure Speech SDK or Gentle
2. Set up SQLite database with schema
3. Create vocabulary importer for HSK/frequency lists
4. Build first learning activity (graded reader)
5. Implement basic spaced repetition
6. Create skill.md with all commands
7. Test with real learning session

## Code Examples

### Pronunciation Analysis with Azure
```python
import azure.cognitiveservices.speech as speechsdk

def analyze_pronunciation(reference_text, audio_file):
    speech_config = speechsdk.SpeechConfig(
        subscription=key, 
        region=region
    )
    
    pronunciation_config = speechsdk.PronunciationAssessmentConfig(
        reference_text=reference_text,
        grading_system=speechsdk.PronunciationAssessmentGradingSystem.HundredMark,
        granularity=speechsdk.PronunciationAssessmentGranularity.Phoneme,
        enable_miscue=True
    )
    
    # Analyze and return scores
    return scores
```

### Spaced Repetition Implementation
```python
def calculate_next_review(familiarity_score, days_since_last):
    if familiarity_score < 0.4:
        return 1  # Review tomorrow
    elif familiarity_score < 0.6:
        return 3  # Review in 3 days
    elif familiarity_score < 0.8:
        return 7  # Review in a week
    else:
        return 14  # Review in 2 weeks
```

This comprehensive system will create an engaging, effective, and personalized Chinese learning experience that adapts to the user's pace and preferences while maintaining the fun and natural acquisition approach of child language learning.
