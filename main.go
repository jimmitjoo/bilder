package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jimmitjoo/bilder/actions"
	sss "github.com/jimmitjoo/bilder/services/aws/s3"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

var (
	region  = os.Getenv("AWS_REGION")
	srcKey  = "inputs/asdsad.jpeg"
	destKey = "outputs/tjenaberra.jpg"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}

	api := iris.New()

	imageAPI := api.Party("/api/images")
	{
		imageAPI.Post("/upload", UploadImage).Name = "api.images.upload"
		imageAPI.Post("/remove-background", RemoveBackground).Name = "api.images.remove-background"
	}

	tmpl := iris.HTML("./views", ".html")
	api.RegisterView(tmpl)

	api.Get("/", Index)
	imageWeb := api.Party("/images")
	{
		imageWeb.Get("/create", CreateImage)
		imageWeb.Get("/show", ShowImage).Name = "web.images.show"
	}

	api.Listen(":8080")

}

func RemoveBackground(ctx iris.Context) {

	// Get the srcLink from the post request
	srcKey := ctx.PostValue("srcKey")

	// srcLink is a signed url, so we need to unescape it
	srcKey, _ = url.QueryUnescape(srcKey)

	// Presign the srcLink
	s3Client, _ := sss.NewClient(region)
	presignClient := s3.NewPresignClient(s3Client)
	presigner := sss.Presigner{
		PresignClient: presignClient,
	}
	inputSrcSigned, _ := presigner.GetObject(srcKey, 3600)

	destKey := fmt.Sprintf("uploads/%s%s", uuid.New().String(), filepath.Ext(srcKey))
	outputSigned, _ := presigner.PutObject(destKey, 3600)
	outputGetterSigned, _ := presigner.GetObject(destKey, 3600)

	actions.RemoveBackground(inputSrcSigned, outputSigned, ctx)

	ctx.Redirect("/images/show?imageUrl=" + url.QueryEscape(outputGetterSigned.URL) + "&destKey=" + destKey)
}

func Index(ctx iris.Context) {

	data := iris.Map{
		"Title": "Public Index Title",
	}

	ctx.ViewLayout("index")
	if err := ctx.View("index", data); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func ShowImage(ctx iris.Context) {

	// the url looks like this: /images/show?imageUrl={signedUrl}
	imageUrl := ctx.URLParam("imageUrl")
	destKey := ctx.URLParam("destKey")

	// Unescape the parameters
	imageUrl, _ = url.QueryUnescape(imageUrl)
	destKey, _ = url.QueryUnescape(destKey)

	ctx.ViewLayout("show-image")
	ctx.ViewData("imageUrl", imageUrl)
	ctx.ViewData("destKey", destKey)

	if err := ctx.View("show-image"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

func CreateImage(ctx iris.Context) {

	// Get the route for uploading an image
	uploadRoute := ctx.Application().GetRouteReadOnly("api.images.upload").Path()

	ctx.ViewLayout("upload-image")
	ctx.ViewData("uploadRoute", uploadRoute)

	if err := ctx.View("upload-image"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}

	// Render a form for the user to upload an image
	/*ctx.HTML(`<form action="` + uploadRoute + `" method="post" enctype="multipart/form-data">
			<input type="file" name="file">
			<input type="submit" value="Upload">
		</form>
	`)*/

}

func UploadImage(ctx iris.Context) {

	file, info, err := ctx.FormFile("file")
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString("file, info, err := ctx.FormFile(\"file\")")
		ctx.WriteString(err.Error())
		return
	}
	defer file.Close()

	// Generate a randomized destination key with the same extension as the uploaded file
	destKey := fmt.Sprintf("uploads/%s%s", uuid.New().String(), filepath.Ext(info.Filename))

	// Create an S3 client
	s3Client, err := sss.NewClient(region)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString("s3Client, err := sss.NewClient(region)")
		ctx.WriteString(err.Error())
		return
	}

	// Create the S3 input parameters
	params := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("AWS_BUCKET")),
		Key:    aws.String(destKey),
		Body:   file,
	}

	// Upload the file to S3
	_, err = s3Client.PutObject(context.TODO(), params)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString("_, err = s3Client.PutObject(context.TODO(), params)")
		ctx.WriteString(err.Error())
		return
	}

	// Generate a presigned URL for the uploaded file
	presignClient := s3.NewPresignClient(s3Client)
	presigner := sss.Presigner{
		PresignClient: presignClient,
	}

	outputLinkSigned, err := presigner.GetObject(destKey, 3600)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.WriteString("outputLinkSigned, err := presigner.GetObject(destKey, 3600)")
		ctx.WriteString(err.Error())
		return
	}

	// Url encode the key
	key := url.QueryEscape(outputLinkSigned.URL)

	// Get the route for showing the image
	showRoute := ctx.Application().GetRouteReadOnly("web.images.show").Path()

	// Redirect to the image show route
	ctx.Redirect(showRoute + "/?imageUrl=" + key + "&destKey=" + destKey)

	// Return the presigned URL
	/*ctx.JSON(iris.Map{
		"link": outputLinkSigned.URL,
	})*/
}
