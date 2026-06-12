import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/core/api/playlists_api.dart';
import 'package:music_room/features/playlist_editor/playlists_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';

class CreatePlaylistScreen extends ConsumerStatefulWidget {
  const CreatePlaylistScreen({super.key});

  @override
  ConsumerState<CreatePlaylistScreen> createState() =>
      _CreatePlaylistScreenState();
}

class _CreatePlaylistScreenState extends ConsumerState<CreatePlaylistScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nameCtrl = TextEditingController();

  String _visibility = 'public';
  int _license = 0;
  bool _submitting = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    super.dispose();
  }

  String _licenseHint() => _license == 0
      ? 'Anyone with access to this playlist can add, remove, and reorder tracks.'
      : 'Only you and collaborators you invite can edit the tracks.';

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _submitting = true);
    try {
      final playlist = await ref.read(playlistsApiProvider).create(
            name: _nameCtrl.text.trim(),
            visibility: _visibility,
            license: _license,
          );
      ref.invalidate(playlistsProvider);
      if (mounted) context.go('/playlists/${playlist.id}');
    } on DioException catch (e) {
      if (!mounted) return;
      final msg = e.response?.data is Map
          ? (e.response?.data['error'] as String?)
          : null;
      AppSnackBar.show(context,
          message: msg ?? 'Could not create the playlist. Try again.',
          type: SnackBarType.error);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Create playlist')),
      body: AbsorbPointer(
        absorbing: _submitting,
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextFormField(
                controller: _nameCtrl,
                textInputAction: TextInputAction.done,
                decoration: const InputDecoration(
                  labelText: 'Playlist name',
                  border: OutlineInputBorder(),
                ),
                validator: (v) =>
                    (v == null || v.trim().isEmpty) ? 'Name is required' : null,
              ),
              const SizedBox(height: 20),
              const Text('Visibility'),
              const SizedBox(height: 8),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'public', label: Text('Public')),
                  ButtonSegment(value: 'private', label: Text('Private')),
                ],
                selected: {_visibility},
                onSelectionChanged: (s) =>
                    setState(() => _visibility = s.first),
              ),
              const SizedBox(height: 6),
              Text(
                _visibility == 'public'
                    ? 'Anyone can find and open this playlist.'
                    : 'Only people you invite can open this playlist.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: 20),
              const Text('License'),
              const SizedBox(height: 8),
              SegmentedButton<int>(
                segments: const [
                  ButtonSegment(value: 0, label: Text('Anyone edits')),
                  ButtonSegment(value: 1, label: Text('Invited only')),
                ],
                selected: {_license},
                onSelectionChanged: (s) => setState(() => _license = s.first),
              ),
              const SizedBox(height: 6),
              Text(
                _licenseHint(),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: 28),
              FilledButton(
                onPressed: _submitting ? null : _submit,
                child: _submitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Create playlist'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
