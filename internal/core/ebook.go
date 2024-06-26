package core

import (
	"os"
	fp "path/filepath"
	"strconv"

	epub "github.com/go-shiori/go-epub"
	"github.com/go-shiori/shiori/internal/dependencies"
	"github.com/go-shiori/shiori/internal/model"
	"github.com/pkg/errors"
)

// GenerateEbook receives a `ProcessRequest` and generates an ebook file in the destination path specified.
// The destination path `dstPath` should include file name with ".epub" extension
// The bookmark model will be used to update the UI based on whether this function is successful or not.
func GenerateEbook(deps *dependencies.Dependencies, req model.EbookProcessRequest) (book model.BookmarkDTO, err error) {
	book = req.Bookmark
	dstPath := model.GetEbookPath(&book)

	// Make sure bookmark ID is defined
	if book.ID == 0 {
		return book, errors.New("bookmark ID is not valid")
	}

	if deps.Domains.Storage.FileExists(dstPath) && req.SkipExisting {
		return book, nil
	}

	// Get current state of bookmark cheak archive and thumb
	strID := strconv.Itoa(book.ID)

	bookmarkThumbnailPath := model.GetThumbnailPath(&book)

	if deps.Domains.Storage.FileExists(bookmarkThumbnailPath) {
		book.ImageURL = fp.Join("/", "bookmark", strID, "thumb")
	}

	// Create temporary epub file
	tmpFile, err := os.CreateTemp("", "ebook")
	if err != nil {
		return book, errors.Wrap(err, "can't create temporary EPUB file")
	}
	defer os.Remove(tmpFile.Name())

	// Create last line of ebook
	lastline := `<hr/><p style="text-align:center">Generated By <a href="https://github.com/go-shiori/shiori">Shiori</a> From <a href="` + book.URL + `">This Page</a></p>`

	// Create ebook
	ebook, err := epub.NewEpub(book.Title)
	if err != nil {
		return book, errors.Wrap(err, "can't create EPUB")
	}

	ebook.SetTitle(book.Title)
	ebook.SetAuthor(book.Author)
	ebook.SetDescription(book.Excerpt)
	_, err = ebook.AddSection(`<h1 style="text-align:center"> `+book.Title+` </h1>`+book.HTML+lastline, book.Title, "", "")
	if err != nil {
		return book, errors.Wrap(err, "can't add ebook Section")
	}
	ebook.EmbedImages()
	err = ebook.Write(tmpFile.Name())
	if err != nil {
		return book, errors.Wrap(err, "can't create ebook file")
	}

	defer tmpFile.Close()

	// If everything go well we move ebook to dstPath
	err = deps.Domains.Storage.WriteFile(dstPath, tmpFile)
	if err != nil {
		return book, errors.Wrap(err, "failed move ebook to destination")
	}

	book.HasEbook = true
	return book, nil
}
