# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Fixed

* streaks will work now

### Added

* basic math command

### Changed

* some command outputs


## [3.1.1] - The release that should be 3.2.0

State files are incompatible with [3.1.0]

### Fixed

* makefile installation path

### Added

* fischPatsch and fishPat statistics
* voicemails aka leave a message for a user


## [3.1.0]

### Added

* makefile for installation

### Changed

* filenames in the log will only show up when using --debug
* timestamp in the log shows date and time to milliseconds
* config file structure
* state file should now be in /var/lib


## [3.0.1]

### Fixed

* Bot detection

### Added

* join and leave commands
* owner only join and leave commands


## [3.0.0]

Working but some features from v1 and v2 are missing:

* the math command
* merlin's spell checker

### Added

* Analytics Log


[Unreleased]: https://github.com/Chronophylos/chb3/compare/v3.2.0..HEAD
[3.2.0]: https://github.com/Chronophylos/chb3/compare/v3.1.4..v3.2.0
[3.1.4]: https://github.com/Chronophylos/chb3/compare/v3.1.3..v3.1.4
[3.1.3]: https://github.com/Chronophylos/chb3/compare/v3.1.2..v3.1.3
[3.1.2]: https://github.com/Chronophylos/chb3/compare/v3.1.1..v3.1.2
[3.1.1]: https://github.com/Chronophylos/chb3/compare/v3.1.0..v3.1.1
[3.1.0]: https://github.com/Chronophylos/chb3/compare/v3.0.1..v3.1.0
[3.0.1]: https://github.com/Chronophylos/chb3/compare/v3.0.0..v3.0.1
[3.0.0]: https://github.com/Chronophylos/chb3/releases/tag/v3.0.0

[//]: # vim: set foldlevel=9:
