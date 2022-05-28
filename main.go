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

	"google.golang.org/api/googleapi"

	"github.com/pkg/errors"
	_ "gomodules.xyz/gdrive-utils"
	gdrive_utils "gomodules.xyz/gdrive-utils"
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

	service, err := drive.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Drive client")

	//_, err = FindParentFolderId(service, fileId)
	//handleError(err, "Error finding parent folder id")

	folderId, err := GetFolderId(service, fileId, "candidates/tamal.saha@gmail.com")
	handleError(err, "Error finding parent folder id")
	fmt.Println(folderId)

	// folders := strings.Split(path.Clean(""), "/")
	// fmt.Println(folders, path.Clean(""))

	// AddPermission(service)
	// ListPlaylistItems(service, "PLoiT1Gv2KR1gc4FN0f7w92RhAHTKbPotT")
}

const fileId = "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"

func AddPermission(svc *drive.Service) {
	expirationTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	p, err := svc.Permissions.Create(fileId, &drive.Permission{
		AllowFileDiscovery:         false,
		Deleted:                    false,
		DisplayName:                "",
		Domain:                     "",
		EmailAddress:               "tamal.saha@gmail.com",
		ExpirationTime:             expirationTime,
		Id:                         "",
		Kind:                       "",
		PendingOwner:               false,
		PermissionDetails:          nil,
		PhotoLink:                  "",
		Role:                       "writer",
		TeamDrivePermissionDetails: nil,
		Type:                       "user",
		View:                       "",
		ServerResponse:             googleapi.ServerResponse{},
		ForceSendFields:            nil,
		NullFields:                 nil,
	}).Fields("id").Do()
	if err != nil {
		panic(err)
	}
	printJSON(p)
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
		files, err := svc.Files.List().Q(q).Spaces("drive").Fields("id").Do()
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
