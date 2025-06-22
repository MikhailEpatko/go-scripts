## What is here
There are 5 different modules (4 applications and 1 package with a common code) that perform one common task - replacing text in JSON files with keys and transferring key-value pairs to a third system.

The keys are formed as a string: the id of the json file + the path to the text in the json file.

Applications receive and modify data through the Web API of the applications whose data they serve.

Each application has its own:

- README.md
- main.go to launch.
The common package contains the configs of all applications, the structures they use, and the general functions of working with the cache.

## How does it work

1) install Go: https://go.dev/doc/install
2) run **from_app_to_json_file**.

```go
go run from_app_to_json_file/main.go
```
It will make json files with key-value pairs for a third system from the application's json files.\
The files will be created in the ./files package.

3) check the contents of the created json files.\
Texts should be specified as values for keys, not other keys.

4) run **update_in_third_system**.

```go
go run update_in_third_system/main.go
```
It will upload the keys to a third system.\
The keys will be loaded into separate sets for each table.

5) check the availability of keys in the third system.

6) if everything is OK in the third system, then run **update_in_app**.
```go
go run update_in_app/main.go

```

It will:

- disable updating the application caches,
- download json files from the application,
- replace texts in json files with keys,
- save new son files through the application API,
- and save them to **<table.name >-new-ids.txt**. The ID of the new json files,
- enable cache updates upon completion of its work.

7) If you need to roll back the changes, then launch **rollback_app**:
```go
go run rollback_app/main.go

```
It will:
- disable cache updates,
- and delete json files through the application API, whose ids it reads from the files **<table.name >-new-ids.txt**,
- enable cache updates.