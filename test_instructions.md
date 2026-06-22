# Create 10 second video for test.ass
```
ffmpeg -f lavfi -i testsrc=duration=10:size=640x360:rate=24 -f lavfi -i "sine=frequency=400:sample_rate=16000:duration=10" \
-af "volume=enable='between(t,1,3) + between(t,5,7)':volume=0dB, volume=enable='not(between(t,1,3) + between(t,5,7))':volume=-90dB" \
-c:v libx264 -c:a aac -pix_fmt yuv420p video.mp4 -y 
```

# Download video and audio for test youtube video
```
yt-dlp -f "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]" "[TARGET_URL] -o youtube_test.mp4
```

# Download english subtitles text track only
```
yt-dlp --write-subs --sub-langs en --skip-download "[TARGET_URL]" -o youtube_test_subs
```

## Download english auto-generated subtitles text track only
```
yt-dlp --write-auto-subs --sub-langs en --skip-download "[TARGET_URL]" -o youtube_test_auto_subs
```

# Convert raw .vtt (Web Video Text Tracks) to .ass (Auto Sub Station Alpha)
```
ffmpeg -i youtube_test_subs.en.vtt youtube_test_subs.ass
```

