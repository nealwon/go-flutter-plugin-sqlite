# go-flutter-plugin-sqlite

This Go package implements the host-side of the Flutter [sqflite](https://pub.dartlang.org/packages/sqflite) plugin.

The plugin is still under development! Using in prod is not recommended!

## Usage

Import as:

```go
import "github.com/nealwon/go-flutter-plugin-sqlite"
```

Then add the following option to your go-flutter [application options](https://github.com/go-flutter-desktop/go-flutter/blob/68868301742b864b719b31ae51c7ec4b3b642d1a/example/simpleDemo/main.go#L53):

```go
flutter.AddPlugin(sqflite.NewSqflitePlugin("myOrganizationOrUsername","myApplicationName")),
```

Change the values of the Vendor and Application names to a custom and unique
string, so it doesn't conflict with other organizations.
