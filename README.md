# 4chan-rss

Returns an RSS feed of a board's threads. Like the official RSS ([index.rss](https://boards.4channel.org/g/index.rss)), but removes threads with less than `-n` number of replies to clean up fetching threads with no replies.

## Boards

- `news` – News  
- `g` – Technology  
- `ck` – Cooking & Food  
- `out` – Outdoors  
- `v` – Video Games  
- `vp` – Pokemon  
- `sp` – Sports  
- `co` – Comics & Cartoons  
- `trv` – Travel  
- `biz` – Business  

## Installation

Make sure you have [Go installed](https://golang.org/dl/). Then clone and build the tool:

```bash
git clone https://github.com/fluteds/4chan-rss.git
cd 4chan-rss
go build -o 4chan-rss
./4chan-rss -b g -n 30 -p 1
```

## Usage

The script can be run from a terminal:

```bash
./4chan-rss -b g,vg,ck -n 30 -p 1 -f general
```

Or you can run it automatically via a GitHub Action and change the boards fetched within the `create-feed` step.

## Arguments

- `-b <boards>` – Boards to fetch, comma-separated (e.g. `g,v,ck`)
- `-n <number>` – Minimum number of replies to include thread (default: `0`)
- `-p <pages>` – Number of pages to parse (default: `1`)
- `-f <"general">` - Filter out keywords from being fetched

## GitHub Action

You can automate the feed generation using GitHub Actions.  
Update the `boards` and `arguments` in the `create-feed` job in your workflow.

> [!NOTE]
> This will create a rss.xml file in your repo

Make sure to build the binary before this step using `go build`.

## Planned Features

- Save output to file
- Filter by thread keyword
