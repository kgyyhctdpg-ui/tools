package sms

import (
	"context"
	"errors"
	"fmt"
)

// Client routes messages to registered providers.
type Client struct {
	defaultProvider Provider
	senders         map[Provider]Sender
}

// NewClient creates a client with senders registered.
func NewClient(senders ...Sender) *Client {
	client := &Client{
		senders: make(map[Provider]Sender),
	}
	for _, sender := range senders {
		client.Use(sender)
	}
	return client
}

// Use registers or replaces a sender.
func (client *Client) Use(sender Sender) {
	if sender == nil {
		return
	}
	if client.senders == nil {
		client.senders = make(map[Provider]Sender)
	}
	provider := sender.Provider()
	client.senders[provider] = sender
	if client.defaultProvider == "" {
		client.defaultProvider = provider
	}
}

// SetDefault sets the provider used by Send.
func (client *Client) SetDefault(provider Provider) error {
	if _, ok := client.senders[provider]; !ok {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}
	client.defaultProvider = provider
	return nil
}

// Send sends with the default provider.
func (client *Client) Send(ctx context.Context, message Message) (*Response, error) {
	if client.defaultProvider == "" {
		return nil, ErrProviderNotFound
	}
	return client.SendWith(ctx, client.defaultProvider, message)
}

// SendWith sends with a specific provider.
func (client *Client) SendWith(ctx context.Context, provider Provider, message Message) (*Response, error) {
	if client == nil || client.senders == nil {
		return nil, ErrProviderNotFound
	}
	sender, ok := client.senders[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}
	return sender.Send(ctx, message)
}

// IsProviderNotFound reports whether err wraps ErrProviderNotFound.
func IsProviderNotFound(err error) bool {
	return errors.Is(err, ErrProviderNotFound)
}
