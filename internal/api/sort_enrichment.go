package api

import "context"

func metadataReadonlyFromHomeOrganization(client HomeOrganizationClient) MetadataReadonlyClient {
	return MetadataReadonlyClient{endpoint: client.endpoint, client: client.client}
}

func (client MetadataReadonlyClient) enrichSortedDeviceRows(ctx context.Context, houseID string, rows []any, credentials MetadataReadonlyCredentials) ([]any, error) {
	if len(rows) == 0 || houseID == "" {
		return rows, nil
	}
	needsLookup := false
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if ok && nodeSortRowID(row) == "" && firstAnyString(row, "name", "alias") != "" {
			needsLookup = true
			break
		}
	}
	if !needsLookup {
		return rows, nil
	}
	entities, err := NewEntityListClient(client.endpoint, client.client).Run(ctx, EntityListRequest{
		HouseID: houseID,
		Credentials: EntityListCredentials{
			Authorization: credentials.Authorization,
			ClientID:      credentials.ClientID,
		},
	})
	if err != nil {
		return rows, err
	}
	byNameRoom := map[string]EntitySummary{}
	for _, entity := range entities.Entities {
		if entity.Type == "device" && entity.Name != "" && entity.ID != "" {
			byNameRoom[entity.Name+"\x00"+entity.RoomID] = entity
		}
	}
	enriched := make([]any, 0, len(rows))
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			enriched = append(enriched, raw)
			continue
		}
		if nodeSortRowID(row) != "" {
			enriched = append(enriched, row)
			continue
		}
		name := firstAnyString(row, "name", "alias")
		roomID := firstAnyString(row, "roomId")
		entity, ok := byNameRoom[name+"\x00"+roomID]
		if !ok {
			enriched = append(enriched, row)
			continue
		}
		copyRow := map[string]any{}
		for key, value := range row {
			copyRow[key] = value
		}
		copyRow["id"] = entity.ID
		copyRow["resId"] = entity.ID
		copyRow["typeId"] = "2"
		enriched = append(enriched, copyRow)
	}
	return enriched, nil
}
