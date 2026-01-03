package ids

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	wordIDRe     = regexp.MustCompile(`^w_[0-9a-f]{12}$`)
	grammarIDRe  = regexp.MustCompile(`^g_[0-9a-f]{12}$`)
	charIDRe     = regexp.MustCompile(`^c_[0-9a-f]{12}$`)
	itemIDRe     = regexp.MustCompile(`^[wgc]_[0-9a-f]{12}$`)
	contentIDRe  = regexp.MustCompile(`^ct_[0-9a-f]{12}$`)
	artifactIDRe = regexp.MustCompile(`^ar_[0-9a-f]{12}$`)
	eventULIDRe  = regexp.MustCompile(`^ev_[0-9A-Z]{26}$`)
	eventUUIDRe  = regexp.MustCompile(`^ev_[0-9a-f]{32}$`)
)

var entropy = ulid.Monotonic(rand.Reader, 0)

func NormalizePinyin(value string) string {
	parts := strings.Fields(value)
	return strings.ToLower(strings.Join(parts, " "))
}

func hex12(payload string) string {
	h := sha1.Sum([]byte(payload))
	return hex.EncodeToString(h[:])[:12]
}

func WordID(hanzi, pinyin string) string {
	normalized := NormalizePinyin(pinyin)
	return "w_" + hex12("word|"+hanzi+"|"+normalized)
}

func GrammarID(grammarKey string) string {
	return "g_" + hex12("grammar|"+grammarKey)
}

func CharID(character string) string {
	return "c_" + hex12("char|"+character)
}

func ContentID(contentType, key string) string {
	return "ct_" + hex12(contentType+"|"+key)
}

func ArtifactID(path string) string {
	return "ar_" + hex12(path)
}

func EventIDULID() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return "ev_" + id.String()
}

func IsWordID(value string) bool {
	return wordIDRe.MatchString(value)
}

func IsGrammarID(value string) bool {
	return grammarIDRe.MatchString(value)
}

func IsCharID(value string) bool {
	return charIDRe.MatchString(value)
}

func IsItemID(value string) bool {
	return itemIDRe.MatchString(value)
}

func IsEventID(value string) bool {
	return eventULIDRe.MatchString(value) || eventUUIDRe.MatchString(value)
}

func IsContentID(value string) bool {
	return contentIDRe.MatchString(value)
}

func IsArtifactID(value string) bool {
	return artifactIDRe.MatchString(value)
}

func ItemType(value string) string {
	if IsWordID(value) {
		return "word"
	}
	if IsGrammarID(value) {
		return "grammar"
	}
	if IsCharID(value) {
		return "character"
	}
	return ""
}
