package cmd

import (
	"context"
	"sync"

	"github.com/chatwoot/chatwoot-cli/internal/agentfmt"
	"github.com/chatwoot/chatwoot-cli/internal/api"
)

type resolvedContact struct {
	Name  string
	Email string
	Phone string
}

func resolveConversationSummaries(ctx context.Context, client *api.Client, summaries []agentfmt.ConversationSummary) []agentfmt.ConversationSummary {
	if len(summaries) == 0 || !flags.ResolveNames {
		return summaries
	}

	inboxIDs := make(map[int]struct{})
	contactIDs := make(map[int]struct{})
	for _, summary := range summaries {
		if summary.InboxID > 0 {
			inboxIDs[summary.InboxID] = struct{}{}
		}
		if summary.ContactID > 0 {
			contactIDs[summary.ContactID] = struct{}{}
		}
	}

	inboxNames := resolveInboxNames(ctx, client, inboxIDs)
	contactMap := resolveContactRefs(ctx, client, contactIDs)

	out := make([]agentfmt.ConversationSummary, len(summaries))
	copy(out, summaries)

	for i := range out {
		summary := &out[i]
		if name, ok := inboxNames[summary.InboxID]; ok {
			summary.Path = applyPathLabel(summary.Path, "inbox", summary.InboxID, name)
		}
		if contact, ok := contactMap[summary.ContactID]; ok {
			if summary.Contact == nil {
				summary.Contact = &agentfmt.ContactRef{
					ID:    summary.ContactID,
					Name:  contact.Name,
					Email: contact.Email,
					Phone: contact.Phone,
				}
			} else {
				if summary.Contact.Name == "" {
					summary.Contact.Name = contact.Name
				}
				if summary.Contact.Email == "" {
					summary.Contact.Email = contact.Email
				}
				if summary.Contact.Phone == "" {
					summary.Contact.Phone = contact.Phone
				}
			}
			summary.Path = applyPathLabel(summary.Path, "contact", summary.ContactID, contact.Name)
		}
	}

	return out
}

func resolveConversationDetail(ctx context.Context, client *api.Client, detail agentfmt.ConversationDetail) agentfmt.ConversationDetail {
	summaries := resolveConversationSummaries(ctx, client, []agentfmt.ConversationSummary{detail.ConversationSummary})
	if len(summaries) == 1 {
		detail.ConversationSummary = summaries[0]
	}
	return detail
}

func resolveInboxNames(ctx context.Context, client *api.Client, inboxIDs map[int]struct{}) map[int]string {
	if len(inboxIDs) == 0 {
		return map[int]string{}
	}

	inboxes, err := client.Inboxes().List(ctx)
	if err != nil {
		return map[int]string{}
	}

	names := make(map[int]string)
	for _, inbox := range inboxes {
		if _, ok := inboxIDs[inbox.ID]; ok {
			names[inbox.ID] = inbox.Name
		}
	}
	return names
}

func resolveContactRefs(ctx context.Context, client *api.Client, contactIDs map[int]struct{}) map[int]resolvedContact {
	if len(contactIDs) == 0 {
		return map[int]resolvedContact{}
	}

	ids := make([]int, 0, len(contactIDs))
	for id := range contactIDs {
		ids = append(ids, id)
	}

	results := make(map[int]resolvedContact)
	var mu sync.Mutex

	// Use 5 workers to parallelize API calls without overwhelming the server.
	// Reduced if fewer contacts to avoid unnecessary goroutines.
	workerCount := 5
	if len(ids) < workerCount {
		workerCount = len(ids)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}
				contact, err := client.Contacts().Get(ctx, id)
				if err != nil || contact == nil {
					continue
				}
				mu.Lock()
				results[id] = resolvedContact{
					Name:  contact.Name,
					Email: contact.Email,
					Phone: contact.PhoneNumber,
				}
				mu.Unlock()
			}
		}()
	}

enqueue:
	for _, id := range ids {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- id:
		}
	}
	close(jobs)
	wg.Wait()

	return results
}

func applyPathLabel(path []agentfmt.PathEntry, kind string, id int, label string) []agentfmt.PathEntry {
	if label == "" || len(path) == 0 {
		return path
	}
	for i, entry := range path {
		if entry.Type == kind && entry.ID == id && entry.Label == "" {
			entry.Label = label
			path[i] = entry
		}
	}
	return path
}
