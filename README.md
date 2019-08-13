# youtube-ar

A frontend instance is deployed at https://yansal-youtube-ar.netlify.com/.

An API instance is deployed at https://youtube-ar-2.herokuapp.com/.

## API setup

* Provision postgresql and redis
* Migrate schema
* Add youtube-dl buildpack
* Set AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET, YOUTUBE_API_KEY config
* Push to heroku with ```git push heroku `git subtree split --prefix api`:master```

## TODO

* frontend: show oembed, filter urls by status, delete url, retry url, download file
* api: retry with tor
