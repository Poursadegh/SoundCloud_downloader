#!/bin/bash

echo "SoundCloud MP3 Downloader"
echo "========================"
echo

if [ -z "$1" ]; then
    echo "Usage: ./download.sh \"SoundCloud_URL\""
    echo "Example: ./download.sh \"https://soundcloud.com/artist/track-name\""
    exit 1
fi

echo "Downloading: $1"
echo

./soundcloud-downloader "$1"

if [ $? -eq 0 ]; then
    echo
    echo "Download completed successfully!"
else
    echo
    echo "Download failed. Please check the error message above."
fi 