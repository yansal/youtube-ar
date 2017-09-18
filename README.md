# youtube-ar

## Requirements

* `tor` and `youtube-dl` in `PATH`
* A PostgreSQL database
* An S3 bucket

## Migrate schema

    createdb youtube-ar && psql youtube-ar < schema.sql

## Start application

    go install
    AWS_SDK_LOAD_CONFIG=1 youtube-ar -bucket <my-s3-bucket>

## TODO(features)

* plug webhooks from youtube, soundcloud, etc.
* allow to directly download video
* more storage services

## TODO(internals)

* package queries and templates (maybe with https://github.com/GeertJohan/go.rice)
* use named parameters in queries
