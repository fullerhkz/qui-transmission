// Copyright (c) 2025-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	internalqbittorrent "github.com/fullerhkz/qui-transmission/internal/qbittorrent"
)

type QBittorrentInfoHandler struct {
	clientPool *internalqbittorrent.ClientPool
}

func NewQBittorrentInfoHandler(clientPool *internalqbittorrent.ClientPool) *QBittorrentInfoHandler {
	return &QBittorrentInfoHandler{
		clientPool: clientPool,
	}
}

// GetQBittorrentAppInfo returns Transmission application version and build information.
func (h *QBittorrentInfoHandler) GetQBittorrentAppInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	instanceIDStr := chi.URLParam(r, "instanceID")
	instanceID, err := strconv.Atoi(instanceIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, "Invalid instance ID")
		return
	}

	client, err := h.clientPool.GetClient(ctx, instanceID)
	if err != nil {
		if respondIfInstanceDisabled(w, err, instanceID, "qbittorrentInfo:getAppInfo") {
			return
		}
		log.Error().Err(err).Int("instanceID", instanceID).Msg("Failed to get client")
		RespondError(w, http.StatusInternalServerError, "Failed to get Transmission client")
		return
	}

	// Get Transmission version and build info.
	appInfo, err := h.getQBittorrentAppInfo(ctx, client)
	if err != nil {
		log.Error().Err(err).Int("instanceID", instanceID).Msg("Failed to get Transmission application info")
		RespondError(w, http.StatusInternalServerError, "Failed to get Transmission application info")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(appInfo)
}

// getQBittorrentAppInfo fetches application info from Transmission RPC.
func (h *QBittorrentInfoHandler) getQBittorrentAppInfo(ctx context.Context, client *internalqbittorrent.Client) (*internalqbittorrent.AppInfo, error) {
	appInfo, err := client.GetAppInfo(ctx)
	if err != nil {
		return nil, err
	}

	if appInfo != nil && appInfo.BuildInfo != nil {
		log.Trace().Msgf("Transmission BuildInfo - App Version: %s, RPC Version: %s, Platform: %s, Library: %s, Qt: %s, Bitness: %d",
			appInfo.Version, appInfo.WebAPIVersion, appInfo.BuildInfo.Platform, appInfo.BuildInfo.Libtorrent, appInfo.BuildInfo.Qt, appInfo.BuildInfo.Bitness)
	}

	return appInfo, nil
}
