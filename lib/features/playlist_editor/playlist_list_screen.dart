import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/core/models/playlist.dart';
import 'package:music_room/features/playlist_editor/playlists_provider.dart';
import 'package:music_room/shared/widgets/state_widgets.dart';

class PlaylistListScreen extends ConsumerStatefulWidget {
  const PlaylistListScreen({super.key});

  @override
  ConsumerState<PlaylistListScreen> createState() => _PlaylistListScreenState();
}

class _PlaylistListScreenState extends ConsumerState<PlaylistListScreen> {
  final _searchCtrl = TextEditingController();
  Timer? _debounce;

  @override
  void dispose() {
    _debounce?.cancel();
    _searchCtrl.dispose();
    super.dispose();
  }

  // Debounced so a request fires only after the user pauses typing (300ms).
  // setState updates the clear button immediately on each keystroke.
  void _onSearchChanged(String value) {
    setState(() {});
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), () {
      ref.read(playlistsProvider.notifier).search(value.trim());
    });
  }

  @override
  Widget build(BuildContext context) {
    final playlistsAsync = ref.watch(playlistsProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Playlists')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/playlists/create'),
        icon: const Icon(Icons.add),
        label: const Text('Create'),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
            child: TextField(
              controller: _searchCtrl,
              onChanged: _onSearchChanged,
              textInputAction: TextInputAction.search,
              decoration: InputDecoration(
                hintText: 'Search playlists',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchCtrl.text.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchCtrl.clear();
                          _onSearchChanged('');
                        },
                      ),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                isDense: true,
              ),
            ),
          ),
          Expanded(
            child: playlistsAsync.when(
              loading: () => const AppLoadingWidget(),
              error: (_, _) => AppErrorWidget(
                message: 'Failed to load playlists. Check your connection.',
                onRetry: () => ref.invalidate(playlistsProvider),
              ),
              data: (playlists) => RefreshIndicator(
                onRefresh: () => ref.read(playlistsProvider.notifier).refresh(),
                child: playlists.isEmpty
                    ? ListView(
                        // ListView keeps pull-to-refresh working on an empty list.
                        children: const [
                          SizedBox(height: 120),
                          AppEmptyStateWidget(
                            icon: Icons.queue_music_outlined,
                            message: 'No playlists found',
                          ),
                        ],
                      )
                    : ListView.separated(
                        itemCount: playlists.length,
                        separatorBuilder: (_, _) => const Divider(height: 1),
                        itemBuilder: (_, i) =>
                            _PlaylistTile(playlist: playlists[i]),
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlaylistTile extends ConsumerWidget {
  const _PlaylistTile({required this.playlist});

  final Playlist playlist;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ownerName = ref.watch(playlistOwnerNameProvider(playlist.ownerId));

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      title: Text(
        playlist.name,
        style: const TextStyle(fontWeight: FontWeight.w600),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SizedBox(height: 2),
          Text(
            'by ${ownerName.maybeWhen(data: (n) => n, orElse: () => '...')}',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 4),
          _LicenseChip(isOpen: playlist.isOpenLicense),
        ],
      ),
      trailing: _VisibilityBadge(isPublic: playlist.isPublic),
      onTap: () => context.push('/playlists/${playlist.id}'),
    );
  }
}

class _VisibilityBadge extends StatelessWidget {
  const _VisibilityBadge({required this.isPublic});

  final bool isPublic;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final bg = isPublic ? scheme.secondaryContainer : scheme.errorContainer;
    final fg = isPublic ? scheme.onSecondaryContainer : scheme.onErrorContainer;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(20),
      ),
      child: Text(
        isPublic ? 'Public' : 'Private',
        style: TextStyle(color: fg, fontSize: 12, fontWeight: FontWeight.w500),
      ),
    );
  }
}

class _LicenseChip extends StatelessWidget {
  const _LicenseChip({required this.isOpen});

  final bool isOpen;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          isOpen ? Icons.public : Icons.lock_outline,
          size: 14,
          color: scheme.outline,
        ),
        const SizedBox(width: 4),
        Text(
          isOpen ? 'Anyone can edit' : 'Invited editors only',
          style: Theme.of(context)
              .textTheme
              .bodySmall
              ?.copyWith(color: scheme.outline),
        ),
      ],
    );
  }
}
