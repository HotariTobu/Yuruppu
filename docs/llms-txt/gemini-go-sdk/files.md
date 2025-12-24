# File Management

## Overview

The Files service enables uploading files to Gemini for use in multimodal prompts. Supports images, videos, audio, and documents.

## Uploading Files

### From File Path

```go
file, err := client.Files.UploadFromPath(
    ctx,
    "/path/to/document.pdf",
    &genai.UploadFileConfig{
        MIMEType:    "application/pdf",
        DisplayName: "User Manual",
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Uploaded: %s\n", file.Name)
fmt.Printf("URI: %s\n", file.URI)
```

### From Bytes

```go
data, err := os.ReadFile("/path/to/image.jpg")
if err != nil {
    log.Fatal(err)
}

file, err := client.Files.UploadFromBytes(
    ctx,
    data,
    &genai.UploadFileConfig{
        MIMEType:    "image/jpeg",
        DisplayName: "User Photo",
    },
)
```

## Using Uploaded Files

### In Text Generation

```go
// Upload file first
file, err := client.Files.UploadFromPath(ctx, "/path/to/video.mp4", &genai.UploadFileConfig{
    MIMEType: "video/mp4",
})
if err != nil {
    log.Fatal(err)
}

// Use in prompt
contents := []*genai.Content{
    {
        Role: genai.RoleUser,
        Parts: []*genai.Part{
            {Text: "What happens in this video?"},
            {FileData: &genai.FileData{
                FileURI:  file.URI,
                MIMEType: "video/mp4",
            }},
        },
    },
}

result, err := client.Models.GenerateContent(ctx, "gemini-3-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Text())
```

## Listing Files

```go
page, err := client.Files.List(ctx, &genai.ListFilesConfig{
    PageSize: 10,
})
if err != nil {
    log.Fatal(err)
}

for _, file := range page.Items {
    fmt.Printf("Name: %s\n", file.Name)
    fmt.Printf("Display Name: %s\n", file.DisplayName)
    fmt.Printf("MIME Type: %s\n", file.MIMEType)
    fmt.Printf("Size: %d bytes\n", file.SizeBytes)
    fmt.Printf("URI: %s\n", file.URI)
    fmt.Println("---")
}
```

## Getting File Details

```go
file, err := client.Files.Get(ctx, fileName, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Name: %s\n", file.Name)
fmt.Printf("State: %v\n", file.State)
fmt.Printf("Create Time: %v\n", file.CreateTime)
fmt.Printf("Update Time: %v\n", file.UpdateTime)
```

## Deleting Files

```go
_, err := client.Files.Delete(ctx, file.Name, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println("File deleted successfully")
```

## File States

Files go through processing states:

```go
const (
    FileStateUnspecified FileState = "STATE_UNSPECIFIED"
    FileStateProcessing  FileState = "PROCESSING"
    FileStateActive      FileState = "ACTIVE"
    FileStateFailed      FileState = "FAILED"
)
```

### Waiting for Processing

```go
file, err := client.Files.UploadFromPath(ctx, videoPath, config)
if err != nil {
    log.Fatal(err)
}

// Wait for file to be ready
for file.State == genai.FileStateProcessing {
    time.Sleep(2 * time.Second)
    file, err = client.Files.Get(ctx, file.Name, nil)
    if err != nil {
        log.Fatal(err)
    }
}

if file.State == genai.FileStateFailed {
    log.Fatal("File processing failed")
}

// Now ready to use
fmt.Println("File ready:", file.URI)
```

## Supported File Types

### Images
- JPEG (`image/jpeg`)
- PNG (`image/png`)
- WebP (`image/webp`)

### Videos
- MP4 (`video/mp4`)
- MPEG (`video/mpeg`)
- MOV (`video/mov`)
- AVI (`video/avi`)
- WebM (`video/webm`)

### Audio
- MP3 (`audio/mp3`)
- WAV (`audio/wav`)
- FLAC (`audio/flac`)

### Documents
- PDF (`application/pdf`)
- Plain text (`text/plain`)

## Inline Data vs File Upload

### Use Inline for Small Files

```go
imageData, _ := os.ReadFile("small-image.jpg")

contents := []*genai.Content{
    {
        Parts: []*genai.Part{
            {Text: "Describe this image"},
            {InlineData: &genai.Blob{
                Data:     imageData,
                MIMEType: "image/jpeg",
            }},
        },
    },
}
```

### Use File Upload for Large Files

```go
// Upload once
file, _ := client.Files.UploadFromPath(ctx, "large-video.mp4", config)

// Reuse in multiple requests
contents := []*genai.Content{
    {
        Parts: []*genai.Part{
            {Text: "Summarize this video"},
            {FileData: &genai.FileData{
                FileURI:  file.URI,
                MIMEType: "video/mp4",
            }},
        },
    },
}
```

## Complete Example

```go
func analyzeDocument(ctx context.Context, client *genai.Client, filePath string) (string, error) {
    // Upload document
    file, err := client.Files.UploadFromPath(ctx, filePath, &genai.UploadFileConfig{
        MIMEType:    "application/pdf",
        DisplayName: filepath.Base(filePath),
    })
    if err != nil {
        return "", fmt.Errorf("upload failed: %w", err)
    }

    // Wait for processing
    for file.State == genai.FileStateProcessing {
        time.Sleep(2 * time.Second)
        file, err = client.Files.Get(ctx, file.Name, nil)
        if err != nil {
            return "", fmt.Errorf("get file failed: %w", err)
        }
    }

    if file.State != genai.FileStateActive {
        return "", fmt.Errorf("file processing failed")
    }

    // Generate summary
    contents := []*genai.Content{
        {
            Parts: []*genai.Part{
                {Text: "Summarize this document in 3 bullet points"},
                {FileData: &genai.FileData{
                    FileURI:  file.URI,
                    MIMEType: "application/pdf",
                }},
            },
        },
    }

    result, err := client.Models.GenerateContent(ctx, "gemini-3-flash", contents, nil)
    if err != nil {
        return "", fmt.Errorf("generation failed: %w", err)
    }

    // Clean up
    client.Files.Delete(ctx, file.Name, nil)

    return result.Text(), nil
}
```

## Best Practices

1. **Upload once, reuse**: Upload files once and reference in multiple requests
2. **Wait for processing**: Check file state before using
3. **Clean up**: Delete files when no longer needed
4. **Use inline for small files**: < 20MB typically
5. **Set display names**: For easier identification
6. **Handle failures**: Check file state after upload
7. **Monitor quota**: Track file storage limits
8. **Use appropriate MIME types**: Ensures proper processing
