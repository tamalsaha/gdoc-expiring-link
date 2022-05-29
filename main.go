// Sample Go code for user authorization

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"google.golang.org/api/sheets/v4"

	"github.com/pkg/errors"
	_ "gomodules.xyz/gdrive-utils"
	gdrive "gomodules.xyz/gdrive-utils"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

func main() {
	client, err := gdrive.DefaultClient(".")
	handleError(err, "Error creating YouTube client")

	svcSheets, err := sheets.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Sheets client")

	configDocId := "1KB_Efi9jQcJ0_tCRF4fSLc6TR7QxaBKg05cKXAwbC9E"
	qaTemplateDocId := "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"
	now := time.Now()
	cfg := QuestionConfig{
		ConfigType:            ConfigTypeQuestion,
		QuestionTemplateDocId: qaTemplateDocId,
		StartDate:             Date{now},
		EndDate:               Date{now.Add(5 * 24 * time.Hour)}, // 3 days
		DurationMinutes:       90,                                // 60 mins
	}
	err = SaveConfig(svcSheets, configDocId, cfg)
	handleError(err, "failed to save config")
}

func main_() {
	client, err := gdrive.DefaultClient(".")
	handleError(err, "Error creating YouTube client")

	svcDrive, err := drive.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Drive client")

	svcDocs, err := docs.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Docs client")

	//_, err = FindParentFolderId(service, fileId)
	//handleError(err, "Error finding parent folder id")

	email := "tamal.saha@gmail.com"

	folderId, err := GetFolderId(svcDrive, fileId, "candidates/"+email)
	handleError(err, "Error finding parent folder id")
	fmt.Println("parent folder id", folderId)

	docId, err := CopyDoc(svcDrive, svcDocs, fileId, folderId, "Account Interview", map[string]string{
		"{{email}}":      email,
		"{{start-time}}": time.Now().Format(time.RFC3339),
		"{{end-time}}":   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
	fmt.Println("user file id", docId)

	// AddPermission(svcDrive, docId, email, "writer")

	err = RevokePermission(svcDrive, docId, email)
	handleError(err, "RevokePermission")
}

const fileId = "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"

func AddPermission(svc *drive.Service, docId string, email string, role string) error {
	_, err := svc.Permissions.Create(docId, &drive.Permission{
		EmailAddress: email,
		Role:         role,
		Type:         "user",
	}).Fields("id").Do()
	return err
}

// https://developers.google.com/youtube/v3/getting-started#partial
// parts vs fields
// parts is top level section
// fields are fields inside that section

func RevokePermission(svc *drive.Service, docId string, email string) error {
	call := svc.Permissions.List(docId)
	call = call.Fields("permissions(id,role,type,emailAddress)")
	var perms []*drive.Permission
	err := call.Pages(context.TODO(), func(resp *drive.PermissionList) error {
		perms = append(perms, resp.Permissions...)
		return nil
	})
	printJSON(perms)

	for _, perm := range perms {
		if perm.EmailAddress == email {
			return svc.Permissions.Delete(docId, perm.Id).Do()
		}
	}
	return err
}

func FindParentFolderId(svc *drive.Service, configDocId string) (string, error) {
	d, err := svc.Files.Get(configDocId).Fields("parents").Do()
	if err != nil {
		return "", err
	}
	return d.Parents[0], nil
}

func GetFolderId(svc *drive.Service, configDocId string, p string) (string, error) {
	parentFolderId, err := FindParentFolderId(svc, configDocId)
	if err != nil {
		return "", errors.Wrap(err, "failed to detect root folder id")
	}
	p = path.Clean(p)
	// empty path (p == "")
	if p == "." {
		return parentFolderId, nil
	}
	folders := strings.Split(p, "/")
	for _, folderName := range folders {
		// https://developers.google.com/drive/api/v3/search-files
		q := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and '%s' in parents", folderName, parentFolderId)
		files, err := svc.Files.List().Q(q).Spaces("drive").Do()
		if err != nil {
			return "", errors.Wrapf(err, "failed to find folder %s inside parent folder %s", folderName, parentFolderId)
		}
		if len(files.Files) > 0 {
			parentFolderId = files.Files[0].Id
		} else {
			// https://developers.google.com/drive/api/v3/folder#java
			folderMetadata := &drive.File{
				Name:     folderName,
				MimeType: "application/vnd.google-apps.folder",
				Parents:  []string{parentFolderId},
			}
			folder, err := svc.Files.Create(folderMetadata).Fields("id").Do()
			if err != nil {
				return "", errors.Wrapf(err, "failed to create folder %s inside parent folder %s", folderName, parentFolderId)
			}
			parentFolderId = folder.Id
		}
	}
	return parentFolderId, nil
}

func CopyDoc(svcDrive *drive.Service, svcDocs *docs.Service, templateDocId string, folderId string, docName string, replacements map[string]string) (string, error) {
	var docId string

	// https://developers.google.com/drive/api/v3/search-files
	q := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.document' and '%s' in parents", docName, folderId)
	files, err := svcDrive.Files.List().Q(q).Spaces("drive").Do()
	if err != nil {
		return "", errors.Wrapf(err, "failed to find doc %s inside parent folder %s", docName, folderId)
	}
	if len(files.Files) > 0 {
		docId = files.Files[0].Id
	} else {
		// https://developers.google.com/docs/api/how-tos/documents#copying_an_existing_document
		copyMetadata := &drive.File{
			Name:    docName,
			Parents: []string{folderId},
		}
		doc, err := svcDrive.Files.Copy(templateDocId, copyMetadata).Fields("id").Do()
		if err != nil {
			return "", errors.Wrapf(err, "failed to copy template doc %s into folder %s", templateDocId, folderId)
		}
		docId = doc.Id

		if len(replacements) > 0 {
			// https://developers.google.com/docs/api/how-tos/merge
			req := &docs.BatchUpdateDocumentRequest{
				Requests: make([]*docs.Request, 0, len(replacements)),
			}
			for k, v := range replacements {
				req.Requests = append(req.Requests, &docs.Request{
					ReplaceAllText: &docs.ReplaceAllTextRequest{
						ContainsText: &docs.SubstringMatchCriteria{
							MatchCase: true,
							Text:      k,
						},
						ReplaceText: v,
					},
				})
			}
			_, err := svcDocs.Documents.BatchUpdate(docId, req).Do()
			if err != nil {
				return "", errors.Wrapf(err, "failed to replace template fields in doc %s", docId)
			}
		}
	}
	return docId, nil
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

const (
	TimestampFormat = "1/2/2006 15:04:05"
	DateFormat      = "1/2/2006"
)

type Timestamp struct {
	time.Time
}

// Convert the internal date as CSV string
func (date *Timestamp) MarshalCSV() (string, error) {
	return date.Time.Format(TimestampFormat), nil
}

// Convert the CSV string as internal date
func (date *Timestamp) UnmarshalCSV(csv string) (err error) {
	date.Time, err = time.Parse(TimestampFormat, csv)
	return err
}

type Date struct {
	time.Time
}

// Convert the internal date as CSV string
func (date *Date) MarshalCSV() (string, error) {
	return date.Time.Format(DateFormat), nil
}

// Convert the CSV string as internal date
func (date *Date) UnmarshalCSV(csv string) (err error) {
	date.Time, err = time.Parse(DateFormat, csv)
	return err
}

type NewsSnippet struct {
	Content   string `json:"content" csv:"Content"`
	StartDate Date   `json:"startDate" csv:"Start Date"`
	EndDate   Date   `json:"endDate" csv:"End Date"`
}

//func (s *Server) RegisterNewsAPI(m *macaron.Macaron) {
//	m.Get("/_/news", func(ctx *macaron.Context, c cache.Cache, log *log.Logger) {
//		key := ctx.Req.URL.Path
//		out := c.Get(key)
//		if out == nil {
//			news, err := s.NextNewsSnippet()
//			if err != nil {
//				ctx.Error(http.StatusInternalServerError, err.Error())
//				return
//			}
//			out = news
//			_ = c.Put(key, out, 60) // cache for 60 seconds
//		} else {
//			log.Println(key, "found")
//		}
//		ctx.JSON(http.StatusOK, out)
//	})
//}

const (
	NewsSnippetSpreadsheetId = ""
	NewsSnippetSheet         = ""
)

func NextNewsSnippet(srvSheets *sheets.Service) (*NewsSnippet, error) {
	now := time.Now()

	reader, err := gdrive.NewRowReader(srvSheets, NewsSnippetSpreadsheetId, NewsSnippetSheet, &gdrive.Predicate{
		Header: "End Date",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				var d Date
				err := d.UnmarshalCSV(v.(string))
				if err != nil {
					return -1, err
				}
				if d.Time.After(now) {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})
	if err == io.EOF {
		return &NewsSnippet{}, nil
	} else if err != nil {
		return nil, err
	}

	snippets := []*NewsSnippet{}
	if err := gocsv.UnmarshalCSV(reader, &snippets); err != nil { // Load clients from file
		return nil, err
	}
	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].EndDate.Before(snippets[j].EndDate.Time)
	})
	for i, s := range snippets {
		if s.EndDate.After(now) {
			snippets = snippets[i:]
			break
		}
	}
	if now.After(snippets[0].StartDate.Time) {
		return snippets[0], nil
	}
	return &NewsSnippet{}, nil
}

// GET Page

type ConfigType string

const (
	ConfigTypeQuestion ConfigType = "QuestionConfig"
)

type IConfigType interface {
	Type() ConfigType
}

type QuestionConfig struct {
	ConfigType            ConfigType `json:"configType" csv:"Config Type"`
	QuestionTemplateDocId string     `json:"questionTemplateDocId" csv:"Question Template Doc Id"`
	StartDate             Date       `json:"startDate" csv:"Start Date"`
	EndDate               Date       `json:"endDate" csv:"End Date"`
	DurationMinutes       int        `json:"durationMinutes"  csv:"Duration Minutes"`
}

var _ IConfigType = QuestionConfig{}

func (q QuestionConfig) Type() ConfigType {
	return q.ConfigType
}

const (
	ProjectConfigSheet = "config"
)

func SaveConfig(srvSheets *sheets.Service, configDocId string, cfg QuestionConfig) error {
	w := gdrive.NewRowWriter(srvSheets, configDocId, ProjectConfigSheet, &gdrive.Predicate{
		Header: "Config Type",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				if v.(string) == string(cfg.Type()) {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})

	data := []*QuestionConfig{
		&cfg,
	}
	return gocsv.MarshalCSV(data, w)
}

/*
func GetPage(srvSheets *sheets.Service, configDocId string) {
	now := time.Now()

	reader, err := gdrive.NewRowReader(srvSheets, configDocId, ProjectConfigSheet, &gdrive.Predicate{
		Header: "Config Type",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				var d Date
				err := d.UnmarshalCSV(v.(string))
				if err != nil {
					return -1, err
				}
				if d.Time.After(now) {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})
	if err == io.EOF {
		return &NewsSnippet{}, nil
	} else if err != nil {
		return nil, err
	}

	snippets := []*NewsSnippet{}
	if err := gocsv.UnmarshalCSV(reader, &snippets); err != nil { // Load clients from file
		return nil, err
	}
	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].EndDate.Before(snippets[j].EndDate.Time)
	})
	for i, s := range snippets {
		if s.EndDate.After(now) {
			snippets = snippets[i:]
			break
		}
	}
	if now.After(snippets[0].StartDate.Time) {
		return snippets[0], nil
	}
	return &NewsSnippet{}, nil
}
*/
