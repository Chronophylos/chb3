// Code generated by "stringer -type=Permission"; DO NOT EDIT.

package actions

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Everyone-0]
	_ = x[Subscriber-1]
	_ = x[Regular-2]
	_ = x[Moderator-3]
	_ = x[Broadcaster-4]
	_ = x[Owner-5]
}

const _Permission_name = "EveryoneSubscriberRegularModeratorBroadcasterOwner"

var _Permission_index = [...]uint8{0, 8, 18, 25, 34, 45, 50}

func (i Permission) String() string {
	if i < 0 || i >= Permission(len(_Permission_index)-1) {
		return "Permission(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Permission_name[_Permission_index[i]:_Permission_index[i+1]]
}
