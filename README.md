# toggl-to-redmine

Script to copy time entries from toggl to redmine.

## Usage

`./toggl_to_redmine [--run] date`

* date : format : `YYYY-MM-DD`
* `--run` : time entries will only be sent to redmine if this flag is activated.

## Environment variables

* `T2R_TOGGL_API_KEY` : your toggl api key
* `T2R_TOGGL_PROJECT_ID` : a projet id on toggl to filter on
* `T2R_REDMINE_ENDPOINT` : the endpoint of your redmine instance
* `T2R_REDMINE_API_KEY` : your redmine api key

## Toggl time entries format

### Basic entry

`10042 description`

The first word of the toggl description must be redmine's issue number.

### Comment

`10042 description - comment`

If there is no dash on the description, the part after the issue number will not be used.
When there is a dash, the part after the dash will be used has comment on redmine's time entry.

### Activity

You must add a tag on toggl time entry that as the exact same value as the activity on redmine.

## Dev

### Build

Run 
```
./build
```
