# What is it

I'm self-learning Japanese. I use [`mpd`](https://wiki.archlinux.org/title/Music_Player_Daemon) with `mpc` to play audio files which I extracted from Japanese movies. Often I need to check out subtitles when I can not clearly understand what I just heard. This tool `srt` plus the helper script `sub`, queries and print the corresponding subtitle at the time of current playing audio.

It assumes your subtitle files are under "$HOME/subtitles/", and have the same base name of the audio files. However, you can modify `sub` script easily as your needs.

# Usage

1. Build `srt` with `go build`.
2. Copy/move `srt` and `sub` to somewhere under you `$PATH`.
3. Run `sub` to print the subtitle at current point:

   ![screen](./screen.png)

# Global offset

Sometimes subtitle file may not exactly match the video, that's why media player like `mpv` has the functionality to adjust subtitle delay.

With `srt`, there are two ways to specify a similar subtitle delay offset:

1. With command line flag: `srt -offset 3500 <srt file>` set a 3500 milliseconds delay offset.
2. Or you can add `OFFSET:3500` as the first line of subtitle file. Offset specfied in the file always override the offset specified by command line flag.
