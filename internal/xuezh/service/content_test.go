package service

import "testing"

func TestAppPutAndGetContentFromBytes(t *testing.T) {
	useServiceTestWorkspace(t)

	put, err := New().PutContent(PutContentOptions{
		ContentType: "story",
		Key:         "tea",
		Filename:    "tea.txt",
		Data:        []byte("茶很好喝"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if put.Data["key"] != "tea" || len(put.Artifacts) != 1 {
		t.Fatalf("unexpected put result: %+v", put)
	}

	got, err := New().GetContent("story", "tea")
	if err != nil {
		t.Fatal(err)
	}
	if got.Data["content_id"] != put.Data["content_id"] {
		t.Fatalf("unexpected get result: %+v", got)
	}
}

func TestAppGetContentBytesReturnsStoredFileBytes(t *testing.T) {
	useServiceTestWorkspace(t)

	if _, err := New().PutContent(PutContentOptions{
		ContentType: "story",
		Key:         "tea",
		Filename:    "tea.txt",
		Data:        []byte("茶很好喝"),
	}); err != nil {
		t.Fatal(err)
	}

	got, data, err := New().GetContentBytes("story", "tea")
	if err != nil {
		t.Fatal(err)
	}
	if got.Data["key"] != "tea" || string(data) != "茶很好喝" {
		t.Fatalf("unexpected content bytes: result=%+v data=%q", got, data)
	}
}
