# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

* simple swearfilter for racist slurs

### Changed

* the delimiter for voicemail recipients to `&&`
* always log with pretty print on
* let golang print panics

## [3.3.2] - 2019-11-29

### Added

* answer regal with lager if nightbot writes it
* scambot reply
* leave multiple voicemails by joining the recipients with `and`

### Changed

* compare users by id not name

## [3.3.1] - 2019-11-22

### Added

* alias quickmafs as math

### Changed

* removed milliseconds from logs
* disabled `^` in `#moondye7`
* increased cooldown for `^`

### Removed

* analytics

### Fixed

* migrated users were missing their ids
* a bug where the state client wrote the total count of patschers to the streak
* the bot not joining any channels on start
* missing space in vanish command
* bot crashing if you check the weather after 12:00
* cooldowns not working



## [3.3.0] - 2019-11-11

### Added

* some aliases in german for state controls
* command to check time

### Changed

* Command#Trigger returns an error with more information than bool
* patsch streaks get reset if you patsch to much

### Fixed

* crash when calling weather with a nonexisting city
* city names getting splitted at umlauts and other special characters
* everyone beeing timedout
* the bot wont repeat `^` from another bot


## [3.2.0]

### Added

* cooldowns for `^` and voicemails

### Changed

* the bot should not crash anymore but log the error instead

### Fixed

* a crash when voicemails are too long
* replaying voicemails didnt show how many
* replaying voicemails respects sleep
* voicemails don't get replayed if your first message is a command


## [3.1.4] - 2019-10-25

### Changed

* users wont get voicemails if they join a channel anymore

### Fixed

* voicemails were case sensitive


## [3.1.3] - 2019-10-22

### Changed

* the bot now enforces fish rules

### Fixed

* the bot now knows the difference between to days when patting


## [3.1.2] - 2019-10-21

### Added

* basic math command

### Changed

* some command outputs

### Fixed

* streaks will work now


## [3.1.1] - The release that should be 3.2.0

State files are incompatible with [3.1.0]

### Added

* fischPatsch and fishPat statistics
* voicemails aka leave a message for a user

### Fixed

* makefile installation path


## [3.1.0]

### Added

* makefile for installation

### Changed

* filenames in the log will only show up when using --debug
* timestamp in the log shows date and time to milliseconds
* config file structure
* state file should now be in /var/lib


## [3.0.1]

### Added

* join and leave commands
* owner only join and leave commands

### Fixed

* Bot detection


## [3.0.0]

Working but some features from v1 and v2 are missing:

* the math command
* merlin's spell checker

### Added

* Analytics Log


[Unreleased]: https://github.com/Chronophylos/chb3/compare/v3.3.2..HEAD
[3.3.2]: https://github.com/Chronophylos/chb3/compare/v3.3.1..v3.3.2
[3.3.1]: https://github.com/Chronophylos/chb3/compare/v3.3.0..v3.3.1
[3.3.0]: https://github.com/Chronophylos/chb3/compare/v3.2.0..v3.3.0
[3.2.0]: https://github.com/Chronophylos/chb3/compare/v3.1.4..v3.2.0
[3.1.4]: https://github.com/Chronophylos/chb3/compare/v3.1.3..v3.1.4
[3.1.3]: https://github.com/Chronophylos/chb3/compare/v3.1.2..v3.1.3
[3.1.2]: https://github.com/Chronophylos/chb3/compare/v3.1.1..v3.1.2
[3.1.1]: https://github.com/Chronophylos/chb3/compare/v3.1.0..v3.1.1
[3.1.0]: https://github.com/Chronophylos/chb3/compare/v3.0.1..v3.1.0
[3.0.1]: https://github.com/Chronophylos/chb3/compare/v3.0.0..v3.0.1
[3.0.0]: https://github.com/Chronophylos/chb3/releases/tag/v3.0.0

[//]: # vim: set foldlevel=9:
