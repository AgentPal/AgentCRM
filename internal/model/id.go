package model

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// 实体类型前缀
const (
	PrefixContact  = "cnt"
	PrefixDeal     = "del"
	PrefixActivity = "act"
	PrefixMemo     = "mem"
	PrefixEvent    = "evt"
	PrefixProposal = "prop"
	PrefixAlert    = "alr"
)

var (
	reNonASCII = regexp.MustCompile(`[^a-z0-9-]+`)
	reDashes   = regexp.MustCompile(`-+`)
	reASCII    = regexp.MustCompile(`[\x20-\x7E]`)
)

// NewID 生成一个带前缀的 ULID。格式：<prefix>_<ULID>。
func NewID(prefix string) string {
	t := time.Now().UTC()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	return fmt.Sprintf("%s_%s", prefix, id.String())
}

// Slugify 将名称转为 slug（用于文件名）。
// "John Smith" → "john-smith", "ABC 科技" → "abc-keji", "张三" → "cnt_<id>"
// 纯中文时返回空字符串，调用方应回退到使用 ID。
func Slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = reNonASCII.ReplaceAllString(slug, "")
	slug = reDashes.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// ContactID 生成一个新的联系人 ID。
func ContactID() string {
	return NewID(PrefixContact)
}

// DealID 生成一个新的商机 ID。
func DealID() string {
	return NewID(PrefixDeal)
}

// ActivityID 生成一个新的活动 ID。
func ActivityID() string {
	return NewID(PrefixActivity)
}

// MemoID 生成一个新的记忆 ID。
func MemoID() string {
	return NewID(PrefixMemo)
}
