import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/api/music_api.dart';
import 'package:music_room/core/models/music_track.dart';

/// Shows a bottom sheet for searching music tracks via the Deezer proxy.
/// Returns the selected [MusicTrack], or null if dismissed.
/// Callers are responsible for doing anything with the result (e.g. posting
/// to an event queue) — the sheet only handles search and selection.
Future<MusicTrack?> showMusicSearchSheet(BuildContext context) {
  return showModalBottomSheet<MusicTrack>(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _MusicSearchSheet(),
  );
}

class _MusicSearchSheet extends ConsumerStatefulWidget {
  const _MusicSearchSheet();

  @override
  ConsumerState<_MusicSearchSheet> createState() => _MusicSearchSheetState();
}

class _MusicSearchSheetState extends ConsumerState<_MusicSearchSheet> {
  final _ctrl = TextEditingController();
  Timer? _debounce;

  bool _loading = false;
  bool _hasSearched = false;
  bool _hasError = false;
  List<MusicTrack> _results = [];

  @override
  void dispose() {
    _debounce?.cancel();
    _ctrl.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    _debounce?.cancel();
    if (value.trim().isEmpty) {
      setState(() {
        _results = [];
        _loading = false;
        _hasSearched = false;
        _hasError = false;
      });
      return;
    }
    setState(() {
      _loading = true;
      _hasError = false;
    });
    _debounce = Timer(
      const Duration(milliseconds: 300),
      () => _fetch(value.trim()),
    );
  }

  Future<void> _fetch(String query) async {
    try {
      final results = await ref.read(musicApiProvider).search(query);
      // Discard if the user already typed something different.
      if (mounted && _ctrl.text.trim() == query) {
        setState(() {
          _results = results;
          _loading = false;
          _hasSearched = true;
          _hasError = false;
        });
      }
    } catch (_) {
      if (mounted && _ctrl.text.trim() == query) {
        setState(() {
          _loading = false;
          _hasError = true;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      initialChildSize: 0.6,
      maxChildSize: 0.92,
      expand: false,
      builder: (_, scrollCtrl) => Column(
        children: [
          const SizedBox(height: 8),
          Container(
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.outlineVariant,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    'Search for a track',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.pop(context),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: TextField(
              controller: _ctrl,
              autofocus: true,
              textInputAction: TextInputAction.search,
              decoration: InputDecoration(
                hintText: 'Artist, song title…',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _ctrl.text.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _ctrl.clear();
                          _onChanged('');
                        },
                      ),
                isDense: true,
                border: const OutlineInputBorder(),
              ),
              onChanged: _onChanged,
            ),
          ),
          Expanded(child: _buildBody(scrollCtrl)),
        ],
      ),
    );
  }

  Widget _buildBody(ScrollController scrollCtrl) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_hasError) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.redAccent),
            const SizedBox(height: 12),
            const Text('Search failed. Try again.'),
          ],
        ),
      );
    }

    if (!_hasSearched) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.music_note_outlined,
                size: 56, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 12),
            Text(
              'Type to search for tracks',
              style: TextStyle(color: Theme.of(context).colorScheme.outline),
            ),
          ],
        ),
      );
    }

    if (_results.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.search_off,
                size: 56, color: Theme.of(context).colorScheme.outline),
            const SizedBox(height: 12),
            Text(
              'No tracks found',
              style: TextStyle(color: Theme.of(context).colorScheme.outline),
            ),
          ],
        ),
      );
    }

    return ListView.builder(
      controller: scrollCtrl,
      itemCount: _results.length,
      itemBuilder: (_, i) => _TrackResultTile(
        track: _results[i],
        onTap: () => Navigator.pop(context, _results[i]),
      ),
    );
  }
}

class _TrackResultTile extends StatelessWidget {
  const _TrackResultTile({required this.track, required this.onTap});

  final MusicTrack track;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      leading: ClipRRect(
        borderRadius: BorderRadius.circular(4),
        child: _Cover(url: track.coverUrl),
      ),
      title: Text(
        track.title,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        track.artist,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      onTap: onTap,
    );
  }
}

class _Cover extends StatelessWidget {
  const _Cover({required this.url});

  final String? url;

  @override
  Widget build(BuildContext context) {
    if (url == null) return _placeholder(context);
    return Image.network(
      url!,
      width: 48,
      height: 48,
      fit: BoxFit.cover,
      errorBuilder: (_, _, _) => _placeholder(context),
    );
  }

  Widget _placeholder(BuildContext context) {
    return Container(
      width: 48,
      height: 48,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: Icon(
        Icons.music_note,
        size: 28,
        color: Theme.of(context).colorScheme.onSurfaceVariant,
      ),
    );
  }
}
