package actions

import (
	"fmt"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/jimmitjoo/bilder/services/adobe"
	"github.com/jimmitjoo/bilder/services/adobe/photoshop"
	"github.com/kataras/iris/v12"
)

func Autotone(inputSrcSigned *v4.PresignedHTTPRequest, outputDestSigned *v4.PresignedHTTPRequest, ctx iris.Context) {

	adobe.CheckAccessToken(ctx)

	resp, err := photoshop.Autotone(inputSrcSigned, outputDestSigned)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Poll status
	err = photoshop.PollStatus(resp)
	if err != nil {
		fmt.Println(err)
		return
	}
}
