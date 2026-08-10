package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestExchangeSignaturePreservesTransfersAndIgnoresParticipantOrder(t *testing.T) {
	t.Parallel()

	itemA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	itemB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	itemC := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	first := Exchange{Participants: []Participant{
		{GivesItemID: itemA, ReceivesItemID: itemB},
		{GivesItemID: itemB, ReceivesItemID: itemC},
		{GivesItemID: itemC, ReceivesItemID: itemA},
	}}
	rotated := Exchange{Participants: []Participant{
		{GivesItemID: itemC, ReceivesItemID: itemA},
		{GivesItemID: itemA, ReceivesItemID: itemB},
		{GivesItemID: itemB, ReceivesItemID: itemC},
	}}
	reversed := Exchange{Participants: []Participant{
		{GivesItemID: itemA, ReceivesItemID: itemC},
		{GivesItemID: itemC, ReceivesItemID: itemB},
		{GivesItemID: itemB, ReceivesItemID: itemA},
	}}

	if first.Signature() != rotated.Signature() {
		t.Fatalf("rotated signature = %q, want %q", rotated.Signature(), first.Signature())
	}
	if first.Signature() == reversed.Signature() {
		t.Fatalf("reversed transfer direction must produce another signature: %q", first.Signature())
	}
}

func TestExchangeCompositionKeyIgnoresOrderAndTransferDirection(t *testing.T) {
	t.Parallel()

	itemA := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	itemB := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	itemC := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

	forward := Exchange{Participants: []Participant{
		{GivesItemID: itemA, ReceivesItemID: itemB},
		{GivesItemID: itemB, ReceivesItemID: itemC},
		{GivesItemID: itemC, ReceivesItemID: itemA},
	}}
	reversed := Exchange{Participants: []Participant{
		{GivesItemID: itemC, ReceivesItemID: itemB},
		{GivesItemID: itemA, ReceivesItemID: itemC},
		{GivesItemID: itemB, ReceivesItemID: itemA},
	}}

	if forward.CompositionKey() != reversed.CompositionKey() {
		t.Fatalf("composition keys differ: %q != %q", forward.CompositionKey(), reversed.CompositionKey())
	}
}
