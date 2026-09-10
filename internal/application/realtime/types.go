package realtime

const (
	ActionCreated              = "created"
	ActionUpdated              = "updated"
	ActionDeleted              = "deleted"
	ActionCurrentClientChanged = "current_client_changed"
	ActionMemberAdded          = "member_added"
	ActionMemberRemoved        = "member_removed"

	EntityUser     = "user"
	EntityClient   = "client"
	EntityCampaign = "campaign"
)

func EventType(entity, action string) string {
	return entity + "." + action
}
