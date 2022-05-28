// Sample Go code for user authorization

package main

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/net/context"
	_ "gomodules.xyz/gdrive-utils"
	gdrive_utils "gomodules.xyz/gdrive-utils"
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
	ctx := context.Background()

	client, err := gdrive_utils.DefaultClient(".", youtube.YoutubeReadonlyScope)
	handleError(err, "Error creating YouTube client")
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	handleError(err, "Error creating YouTube client")

	// channelsListByUsername(service, strings.Split("snippet,contentDetails,statistics", ","), channelID)

	ListPlaylists(service, channelID)
	// ListPlaylistItems(service, "PLoiT1Gv2KR1gc4FN0f7w92RhAHTKbPotT")
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
