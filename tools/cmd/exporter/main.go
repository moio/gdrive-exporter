package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/rancherlabs/architecture/internal/drive_auth"
	"github.com/rancherlabs/architecture/internal/utils"
	"google.golang.org/api/drive/v3"

	"github.com/spf13/cobra"
)

var clientSecretPath string
var clientTokenPath string
var folderId string
var destinationPath string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "exporter",
		Short: "Download Google Documents converting them to Office formats",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&clientSecretPath, "client-secret", "s", "", "Path to the client secret file")
	rootCmd.Flags().StringVarP(&clientTokenPath, "client-token-dir", "t", "", "Path to the directory where to store tokens")
	rootCmd.Flags().StringVarP(&folderId, "folder-id", "i", "", "ID of the folder to download")
	rootCmd.Flags().StringVarP(&destinationPath, "destination", "o", "", "Destination path for the downloaded files")

	rootCmd.MarkFlagRequired("client-secret")
	rootCmd.MarkFlagRequired("client-token-dir")
	rootCmd.MarkFlagRequired("folder-id")
	rootCmd.MarkFlagRequired("destination")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	// get Drive service
	ctx, cancel := utils.NewSignalCancelingContext()
	defer cancel()
	srv, err := drive_auth.GetService(ctx, clientSecretPath, clientTokenPath, true, false)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// list all files starting from folderId
	err = downloadAll(folderId, destinationPath, srv, ctx)
	if err != nil {
		log.Fatalf("Unable to download files: %v", err)
	}
}

func downloadAll(downloadId string, destinationPath string, srv *drive.Service, ctx context.Context) error {
	// create destinationPath if it doesn't exist
	err := os.MkdirAll(destinationPath, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create download directory: %v", err)
	}

	query := fmt.Sprintf("'%s' in parents and trashed = false", downloadId)

	err = srv.Files.List().
		Corpora("allDrives").
		IncludeItemsFromAllDrives(true).
		SupportsAllDrives(true).
		Q(query).
		Fields("nextPageToken, files(id, name, parents, mimeType, capabilities)").
		Pages(ctx, func(r *drive.FileList) error {
			for _, i := range r.Files {
				fmt.Printf("Processing: \"%s\" (%v)\n", strings.Join(i.Parents, "/")+"/"+i.Id, i.Name)
				if i.MimeType == "application/vnd.google-apps.folder" {
					destinationPath := path.Join(destinationPath, cleanName(i.Name))
					fmt.Printf("  -> creating directory: %s\n", destinationPath)
					err := downloadAll(i.Id, destinationPath, srv, ctx)
					if err != nil {
						return err
					}
				} else if i.MimeType == "application/vnd.google-apps.document" {
					destinationName := cleanName(i.Name)
					fullPath := path.Join(destinationPath, destinationName+".docx")
					fmt.Printf("  -> downloading: %s\n", fullPath)
					err := downloadDocument(i.Id, fullPath, srv, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
					if err != nil {
						return err
					}
				} else if i.MimeType == "application/vnd.google-apps.spreadsheet" {
					destinationName := cleanName(i.Name)
					fullPath := path.Join(destinationPath, destinationName+".xlsx")
					fmt.Printf("  -> downloading: %s\n", fullPath)
					err := downloadDocument(i.Id, fullPath, srv, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
					if err != nil {
						return err
					}
				} else if i.MimeType == "application/vnd.google-apps.presentation" {
					destinationName := cleanName(i.Name)
					fullPath := path.Join(destinationPath, destinationName+".pptx")
					fmt.Printf("  -> downloading: %s\n", fullPath)
					err := downloadDocument(i.Id, fullPath, srv, "application/vnd.openxmlformats-officedocument.presentationml.presentation")
					if err != nil {
						return err
					}
				} else {
					fmt.Printf("  -> skipping: %s (mime-type %v)\n", destinationPath, i.MimeType)
				}
			}
			return nil
		})
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	return nil
}

// replaces filename weird characters
func cleanName(name string) string {
	forbiddenChars := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, char := range forbiddenChars {
		name = strings.ReplaceAll(name, char, "_")
	}
	return strings.Trim(name, " ")
}

func downloadDocument(id string, name string, srv *drive.Service, exportMimeType string) error {
	resp, err := srv.Files.Export(id, exportMimeType).Download()
	if err != nil {
		return fmt.Errorf("unable to download file: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("unable to create file: %v", err)
	}
	_, err = f.ReadFrom(resp.Body)
	err2 := f.Close()

	if err != nil || err2 != nil {
		return fmt.Errorf("unable to write file: %v, %v", err, err)
	}

	return nil
}
