# Helper Functions

```go
// Create pointer for optional fields
genai.Ptr[float32](0.7)
genai.Ptr[int32](42)

// Quick text content creation
genai.Text("Your prompt") // Returns []*genai.Content

// Create Parts
genai.NewPartFromText("text")
genai.NewPartFromBytes(data, "image/jpeg")
genai.NewPartFromURI("gs://bucket/file.pdf", "application/pdf")
```
