package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/pkg/errors"
	csvtypes "gomodules.xyz/encoding/csv/types"
	_ "gomodules.xyz/gdrive-utils"
	gdrive "gomodules.xyz/gdrive-utils"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func handleError(err error, message string) {
	if message == "" {
		message = "Error making API call"
	}
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Date/Date
//func main() {
//	now := time.Now().UTC()
//	fmt.Println(now.Format(time.RFC3339))
//}

func main() {
	client, err := gdrive.DefaultClient(".")
	handleError(err, "Error creating YouTube client")

	svcSheets, err := sheets.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Sheets client")

	svcDrive, err := drive.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Drive client")

	svcDocs, err := docs.NewService(context.TODO(), option.WithHTTPClient(client))
	handleError(err, "Error creating Docs client")

	email := "tamal.saha@gmail.com"

	configDocId := "1KB_Efi9jQcJ0_tCRF4fSLc6TR7QxaBKg05cKXAwbC9E"
	//qaTemplateDocId := "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"
	//now := time.Now()
	//cfg := QuestionConfig{
	//	ConfigType:            ConfigTypeQuestion,
	//	QuestionTemplateDocId: qaTemplateDocId,
	//	StartDate:             Date{now},
	//	EndDate:               Date{now.Add(5 * 24 * time.Hour)}, // 3 days
	//	Duration:              Duration{90 * time.Minute},        // 60 mins
	//}
	//err = SaveConfig(svcSheets, configDocId, cfg)
	//handleError(err, "failed to save config")

	// cfg, err := LoadConfig(svcSheets, configDocId)
	// handleError(err, "failed to save config")
	// printJSON(cfg)

	PostPage(svcDrive, svcDocs, svcSheets, configDocId, email)
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

	folderId, err := gdrive.GetFolderId(svcDrive, fileId, "candidates/"+email)
	handleError(err, "Error finding parent folder id")
	fmt.Println("parent folder id", folderId)

	now := time.Now()
	docId, err := gdrive.CopyDoc(svcDrive, svcDocs, fileId, folderId, "Account Interview", map[string]string{
		"{{email}}":      email,
		"{{start-time}}": now.Format(time.RFC3339),
		"{{end-time}}":   now.Add(1 * time.Hour).Format(time.RFC3339),
	})
	fmt.Println("user file id", docId)

	// AddPermission(svcDrive, docId, email, "writer")

	err = gdrive.RevokePermission(svcDrive, docId, email)
	handleError(err, "RevokePermission")
}

const fileId = "16Ff6Lum3F6IeyAEy3P5Xy7R8CITIZRjdwnsRwBg9rD4"

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

type ConfigType string

const (
	ConfigTypeQuestion ConfigType = "QuestionConfig"
)

type QuestionConfig struct {
	ConfigType            ConfigType        `json:"configType" csv:"Config Type"`
	QuestionTemplateDocId string            `json:"questionTemplateDocId" csv:"Question Template Doc Id"`
	StartDate             csvtypes.Date     `json:"startDate" csv:"Start Date"`
	EndDate               csvtypes.Date     `json:"endDate" csv:"End Date"`
	Duration              csvtypes.Duration `json:"duration"  csv:"Duration"`
}

const (
	ProjectConfigSheet = "config"
	ProjectTestSheet   = "test"
)

func SaveConfig(svcSheets *sheets.Service, configDocId string, cfg QuestionConfig) error {
	w := gdrive.NewRowWriter(svcSheets, configDocId, ProjectConfigSheet, &gdrive.Predicate{
		Header: "Config Type",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				if v.(string) == string(cfg.ConfigType) {
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

func LoadConfig(svcSheets *sheets.Service, configDocId string) (*QuestionConfig, error) {
	r, err := gdrive.NewRowReader(svcSheets, configDocId, ProjectConfigSheet, &gdrive.Predicate{
		Header: "Config Type",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				if v.(string) == string(ConfigTypeQuestion) {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})
	if err == io.EOF {
		return nil, errors.New("Question Config not found!")
	} else if err != nil {
		return nil, err
	}

	configs := []*QuestionConfig{}
	if err := gocsv.UnmarshalCSV(r, &configs); err != nil { // Load clients from file
		return nil, err
	}
	return configs[0], nil
}

type TestAnswer struct {
	Email     string             `json:"email" csv:"Email"`
	DocId     string             `json:"docId"  csv:"Doc Id"`
	StartDate csvtypes.Timestamp `json:"startDate" csv:"Start Date"`
	EndDate   csvtypes.Timestamp `json:"endDate" csv:"End Date"`
	Revoked   bool               `json:"revoked"  csv:"Revoked"`
}

func SaveTestAnswer(svcSheets *sheets.Service, configDocId string, ans TestAnswer) error {
	w := gdrive.NewRowWriter(svcSheets, configDocId, ProjectTestSheet, &gdrive.Predicate{
		Header: "Email",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				if v.(string) == ans.Email {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})

	data := []*TestAnswer{
		&ans,
	}
	return gocsv.MarshalCSV(data, w)
}

func LoadTestAnswer(svcSheets *sheets.Service, configDocId, email string) (*TestAnswer, error) {
	r, err := gdrive.NewRowReader(svcSheets, configDocId, ProjectTestSheet, &gdrive.Predicate{
		Header: "Email",
		By: func(column []interface{}) (int, error) {
			for i, v := range column {
				if v.(string) == email {
					return i, nil
				}
			}
			return -1, io.EOF
		},
	})
	//if err == io.EOF {
	//	return nil, errors.Errorf("%s has not started the test yet!", email)
	//} else
	if err != nil {
		return nil, err
	}

	answers := []*TestAnswer{}
	if err := gocsv.UnmarshalCSV(r, &answers); err != nil {
		return nil, err
	}
	return answers[0], nil
}

func GetTestPage(svcSheets *sheets.Service, configDocId string) {
	cfg, err := LoadConfig(svcSheets, configDocId)
	if err != nil {
		panic(err)
	}
	if time.Now().After(cfg.EndDate.Time) {
		panic("Time passed for this test")
	}
	fmt.Printf("%s left to take the test!", time.Until(cfg.EndDate.Time))
}

func PostPage(svcDrive *drive.Service, svcDocs *docs.Service, svcSheets *sheets.Service, configDocId, email string) {
	// already submitted
	// started and x min left to finish the test, redirect, embed
	// did not start, copy file, stat clock

	now := time.Now()

	cfg, err := LoadConfig(svcSheets, configDocId)
	if err != nil {
		panic(err)
	}

	if now.After(cfg.EndDate.Time) {
		panic("Time passed for this test")
	}
	ans, err := LoadTestAnswer(svcSheets, configDocId, email)
	if err != nil && err != io.EOF {
		panic(err) // some error
	}
	if err == nil {
		if now.After(ans.EndDate.Time) {
			panic(fmt.Sprintf("%s passed after test has ended!", time.Since(ans.EndDate.Time)))
		}
	} else {
		ans = &TestAnswer{
			Email:     email,
			DocId:     "",
			StartDate: csvtypes.Timestamp{now},
			EndDate:   csvtypes.Timestamp{now.Add(cfg.Duration.Duration)},
		}

		folderId, err := gdrive.GetFolderId(svcDrive, configDocId, path.Join("candidates", email))
		if err != nil {
			panic(err)
		}
		docName := fmt.Sprintf("%s - Test %s", email, ans.StartDate.Format("2006-01-02"))
		docId, err := gdrive.CopyDoc(
			svcDrive, svcDocs, cfg.QuestionTemplateDocId, folderId, docName, map[string]string{
				"{{email}}":      email,
				"{{start-time}}": ans.StartDate.Format(time.RFC3339),
				"{{end-time}}":   ans.EndDate.Format(time.RFC3339),
			})
		if err != nil {
			panic(err)
		}
		ans.DocId = docId

		err = SaveTestAnswer(svcSheets, configDocId, *ans)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("%s left to take the test!\n", time.Until(ans.EndDate.Time))
	fmt.Printf("https://docs.google.com/document/d/%s/edit\n", ans.DocId)
}
