package schema

func init() {
	registerConversation()
	registerContact()
	registerMessage()
	registerInbox()
	registerAgent()
	registerTeam()
	registerLabel()
}

func registerConversation() {
	Register("conversation", Object(
		"A Chatwoot conversation between a contact and agents",
		map[string]*Schema{
			"id":         Int("Unique conversation identifier"),
			"display_id": Int("Human-friendly conversation identifier"),
			"inbox_id":   Int("ID of the inbox this conversation belongs to"),
			"contact_id": Int("ID of the contact in this conversation"),
			"status": Enum("Current conversation status",
				"open", "resolved", "pending", "snoozed"),
			"priority": Enum("Conversation priority level",
				"urgent", "high", "medium", "low", "none"),
			"assignee_id":       Int("ID of the assigned agent (null if unassigned)"),
			"team_id":           Int("ID of the assigned team (null if unassigned)"),
			"muted":             Bool("Whether the conversation is muted"),
			"unread_count":      Int("Unread message count"),
			"labels":            Array(String("Label name"), "Labels attached to the conversation"),
			"meta":              Map("Conversation metadata"),
			"custom_attributes": Map("Custom attributes for the conversation"),
			"created_at":        Timestamp("When the conversation was created"),
			"last_activity_at":  Timestamp("When the conversation last had activity"),
		},
		"id", "inbox_id", "status", "created_at",
	))
}

func registerContact() {
	Register("contact", Object(
		"A Chatwoot contact representing a customer",
		map[string]*Schema{
			"id":                Int("Unique contact identifier"),
			"name":              String("Contact display name"),
			"email":             String("Contact email address"),
			"phone_number":      String("Contact phone number"),
			"identifier":        String("External identifier for the contact"),
			"thumbnail":         String("Contact avatar thumbnail URL"),
			"custom_attributes": Map("Custom attributes for the contact"),
			"created_at":        Timestamp("When the contact was created"),
			"last_activity_at":  Timestamp("When the contact last had activity"),
		},
		"id", "name", "created_at",
	))
}

func registerMessage() {
	Register("message", Object(
		"A message within a Chatwoot conversation",
		map[string]*Schema{
			"id":              Int("Unique message identifier"),
			"content":         String("Message text content"),
			"content_type":    String("Content type (text, input_text, etc.)"),
			"message_type":    Enum("Type of message", "incoming", "outgoing", "activity", "template"),
			"private":         Bool("Whether this is a private note (not visible to contact)"),
			"conversation_id": Int("ID of the conversation this message belongs to"),
			"sender_id":       Int("ID of the message sender (agent or contact)"),
			"sender_type":     String("Type of sender (User or Contact)"),
			"attachments":     Array(Map("Attachment fields"), "Message attachments"),
			"created_at":      Timestamp("When the message was created"),
		},
		"id", "content", "message_type", "conversation_id", "created_at",
	))
}

func registerInbox() {
	Register("inbox", Object(
		"A Chatwoot inbox representing a communication channel",
		map[string]*Schema{
			"id":                     Int("Unique inbox identifier"),
			"name":                   String("Inbox display name"),
			"channel_type":           String("Type of channel (web, email, api, twitter, etc.)"),
			"avatar_url":             String("Inbox avatar URL"),
			"website_url":            String("Associated website URL"),
			"greeting_enabled":       Bool("Whether greeting message is enabled"),
			"greeting_message":       String("Greeting message text"),
			"enable_auto_assignment": Bool("Whether auto-assignment is enabled"),
		},
		"id", "name", "channel_type",
	))
}

func registerAgent() {
	Register("agent", Object(
		"A Chatwoot agent (user) who handles conversations",
		map[string]*Schema{
			"id":                  Int("Unique agent identifier"),
			"name":                String("Agent display name"),
			"email":               String("Agent email address"),
			"role":                Enum("Agent role", "agent", "administrator"),
			"availability_status": Enum("Current availability status", "online", "busy", "offline"),
			"thumbnail":           String("Agent avatar thumbnail URL"),
			"confirmed_at":        Timestamp("When the agent was confirmed"),
		},
		"id", "name", "email", "role",
	))
}

func registerTeam() {
	Register("team", Object(
		"A Chatwoot team for grouping agents",
		map[string]*Schema{
			"id":                Int("Unique team identifier"),
			"name":              String("Team name"),
			"description":       String("Team description"),
			"allow_auto_assign": Bool("Whether auto-assignment is enabled"),
			"account_id":        Int("Account ID for the team"),
		},
		"id", "name",
	))
}

func registerLabel() {
	Register("label", Object(
		"A Chatwoot label for categorizing conversations",
		map[string]*Schema{
			"id":              Int("Unique label identifier"),
			"title":           String("Label title/name"),
			"color":           String("Label color in hex format (e.g., #FF0000)"),
			"description":     String("Label description"),
			"show_on_sidebar": Bool("Whether label appears in sidebar"),
		},
		"id", "title",
	))
}
