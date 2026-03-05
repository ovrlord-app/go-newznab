# Go Newznab API Client

> newznab/torznab XML API client for Go (golang)

Forked from [go-newznab](https://github.com/mrobinsn/go-newznab) to enhance and modernize where applicable.

## Features
- TV and Movie search
- Search for files with category(s) and query
- Get comments for a NZB
- Get NZB download URL
- Download NZB
- Get latest releases via RSS

## 📚 Guides & Documentation

- 🤝 [Contributing Guide](docs/CONTRIBUTING.md)
- 🔒 [Security Policy](docs/SECURITY.md)

### Example use

```bash
go get github.com/ovrlord-app/go-nzbparser
```

### Initialize a client:
```
client := newznab.New("http://my-usenet-indexer", "my-api-key", 1234, false)

```
Note the missing `/api` part of the URL. Depending on the called method either `/api` or `/rss` will be appended to the given base URL. A valid user ID is only required for RSS methods.

### Get the capabilities of your tracker
```
caps, _ := client.Capabilities()
```
You will want to check the result of this to determine if your tracker supports searching by tvrage, imdb, tvmaze, etc.

### Search using a tvrage id:
```
categories := []int{
    newznab.CategoryTVHD,
    newznab.CategoryTVSD,
}
results, _ := client.SearchWithTVRage(categories, 35048, 3, 1)
```

### Search using an imdb id:
```
categories := []int{
    newznab.CategoryMovieHD,
    newznab.CategoryMovieBluRay,
}
results, _ := client.SearchWithIMDB(categories, "0364569")
```

### Search using a tvmaze id:
```
categories := []int{
    newznab.CategoryTVHD,
    newznab.CategoryTVSD,
}
results, _ := client.SearchWithTVMaze(categories, 80, 3, 1)
```

### Search using a name and set of categories:
```
results, _ := client.SearchWithQueries(categories, "Oldboy", "movie")
```

### Get latest releases for set of categories:
```
results, _ := client.SearchWithQuery(categories, "", "movie")
```

### Load latest releases via RSS:
```
results, _ := client.LoadRSSFeed(categories, 50)
```

### Load latest releases via RSS up to a given NZB id:
```
results, _ := client.LoadRSSFeedUntilNZBID(categories, 50, "nzb-guid", 15)
```

## License

This project is licensed under the MIT - see the [LICENSE](LICENSE) file for details.

## Support

- 🐛 [Issues](https://github.com/ovrlord-app/go-newznab/issues)