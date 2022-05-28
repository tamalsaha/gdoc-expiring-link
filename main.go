// Sample Go code for user authorization

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	_ "gomodules.xyz/gdrive-utils"
	gdrive_utils "gomodules.xyz/gdrive-utils"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

// https://developers.google.com/youtube/v3/guides/working_with_channel_ids
// Use channel id
// Youtube Studio > Customization > Basic Info
func channelsListByUsername(service *youtube.Service, parts []string, channelID string) {
	call := service.Channels.List(parts)
	call = call.Id(channelID)
	response, err := call.Do()
	handleError(err, "")

	fmt.Println(fmt.Sprintf("This channel's ID is %s. Its title is '%s', "+
		"and it has %d views.",
		response.Items[0].Id,
		response.Items[0].Snippet.Title,
		response.Items[0].Statistics.ViewCount))
}

// https://developers.google.com/youtube/v3/getting-started#partial
// parts vs fields
// parts is top level section
// fields are fields inside that section

const channelID = "UCxObRDZ0DtaQe_cCP-dN-xg"

func main() {
	client, err := gdrive_utils.DefaultClient(".", youtube.YoutubeReadonlyScope)
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
	fmt.Println(folderId)

	docId, err := CopyDoc(svcDrive, svcDocs, fileId, folderId, "Account Interview", map[string]string{
		"{{email}}":      email,
		"{{start-time}}": time.Now().Format(time.RFC3339),
		"{{end-time}}":   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	})
	fmt.Println("user file id", docId)

	AddPermission(svcDrive, docId, email, "writer")
	// ListPlaylistItems(service, "PLoiT1Gv2KR1gc4FN0f7w92RhAHTKbPotT")

	// RevokePermission(service, fileId, "tamal.saha@gmail.com")
}

const fileId = "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"

func AddPermission(svc *drive.Service, docId string, email string, role string) error {
	// expirationTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	_, err := svc.Permissions.Create(docId, &drive.Permission{
		EmailAddress: email,
		Role:         role,
		Type:         "user",
	}).Fields("id").Do()
	return err
}

func RevokePermission(svc *drive.Service, docId string, email string) error {
	call := svc.Permissions.List(docId)
	var perms []*drive.Permission
	err := call.Pages(context.TODO(), func(resp *drive.PermissionList) error {
		perms = append(perms, resp.Permissions...)
		return nil
	})
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
	// https://developers.google.com/docs/api/how-tos/documents#copying_an_existing_document
	copyMetadata := &drive.File{
		Name:    docName,
		Parents: []string{folderId},
	}
	doc, err := svcDrive.Files.Copy(templateDocId, copyMetadata).Fields("id", "parents").Do()
	if err != nil {
		return "", errors.Wrapf(err, "failed to copy template doc %s into folder %s", templateDocId, folderId)
	}
	fmt.Println("doc id:", doc.Id)

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
		_, err := svcDocs.Documents.BatchUpdate(doc.Id, req).Do()
		if err != nil {
			return "", errors.Wrapf(err, "failed to replace template fields in doc %s", doc.Id)
		}
	}
	return doc.Id, nil
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

// playlist
// playlistItem
// thumbnail
// video

// https://developers.google.com/youtube/v3/guides/implementation/playlists
func ListPlaylists(service *youtube.Service, channelID string) ([]*youtube.Playlist, error) {
	call := service.Playlists.List(strings.Split("snippet,contentDetails,status", ","))
	// call = call.Fields("items(id,snippet(title,description,publishedAt,tags,thumbnails(high)),contentDetails,status)")
	call = call.ChannelId(channelID)

	var out []*youtube.Playlist
	err := call.Pages(context.TODO(), func(resp *youtube.PlaylistListResponse) error {
		out = append(out, resp.Items...)
		return nil
	})
	return out, err
}

// https://developers.google.com/youtube/v3/docs/playlistItems/list
func ListPlaylistItems(service *youtube.Service, playlistID string) ([]*youtube.PlaylistItem, error) {
	call := service.PlaylistItems.List(strings.Split("snippet,contentDetails,status", ","))
	call = call.Fields("items(snippet(title,description,position,thumbnails(high)),contentDetails,status)")
	call = call.PlaylistId(playlistID)

	var out []*youtube.PlaylistItem
	err := call.Pages(context.Background(), func(resp *youtube.PlaylistItemListResponse) error {
		out = append(out, resp.Items...)
		return nil
	})
	return out, err
}