// Sample Go code for user authorization

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/googleapi"
	"log"
	"strings"
	"time"

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

	AddPermission(service)
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
	}).Do()
	if err != nil {
		panic(err)
	}
	data, _ := json.MarshalIndent(p, "", "  ")
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
