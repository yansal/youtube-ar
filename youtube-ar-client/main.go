package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	var (
		addr       = flag.String("addr", "http://localhost:8080", "youtube-ar listening address")
		s3bucket   = flag.String("bucket", "", "s3 bucket containing the urls to post to youtube-ar")
		lambdafunc = flag.String("lambda", "", "lambda function to invoke when youtube-ar is idle")
	)
	flag.Parse()

	if *lambdafunc == "" || *s3bucket == "" {
		fmt.Fprintln(os.Stderr, "bucket and lambda flags must be set")
		flag.Usage()
		os.Exit(2)
	}

	sess := session.Must(session.NewSession())
	statusURL := *addr + "/_status/"

	// Wait for youtube-ar to be idle
	wait(statusURL)

	// List all objects in s3
	s3svc := s3.New(sess)
	list, err := s3svc.ListObjects(&s3.ListObjectsInput{Bucket: s3bucket})
	if err != nil {
		log.Fatal(err)
	}
	// For each object, post to youtube-ar and delete the object
	for _, obj := range list.Contents {
		if err := post(*addr, *obj.Key); err != nil {
			log.Fatal(err)
		}
		if _, err := s3svc.DeleteObject(&s3.DeleteObjectInput{Bucket: s3bucket, Key: obj.Key}); err != nil {
			log.Fatal(err)
		}
	}

	// Wait for youtube-ar to be idle again
	wait(statusURL)

	// Invoke lambda
	if _, err := lambda.New(sess).Invoke(&lambda.InvokeInput{
		FunctionName: lambdafunc,
	}); err != nil {
		log.Fatal(err)
	}
}

func wait(url string) {
	for {
		err := idle(url)
		if err == nil {
			return
		}
		log.Print(err)
		time.Sleep(time.Second)
	}
}

func idle(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "idle" {
		return fmt.Errorf("youtube-ar is %q", body)
	}

	return nil
}

func post(addr, payload string) error {
	v := url.Values{}
	v.Set("url", payload)
	resp, err := http.PostForm(addr, v)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status %v", resp.StatusCode)
	}
	return nil
}
