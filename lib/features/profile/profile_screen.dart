import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/models/user_profile.dart';
import 'package:music_room/features/profile/profile_provider.dart';

class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(myProfileProvider);

    if (profileAsync.isLoading && profileAsync.valueOrNull == null) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (profileAsync.hasError && profileAsync.valueOrNull == null) {
      return Scaffold(
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                'Failed to load profile',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () => ref.invalidate(myProfileProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
      );
    }

    final profile = profileAsync.requireValue;

    return Scaffold(
      appBar: AppBar(title: const Text('My Profile')),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(myProfileProvider);
          await ref.read(myProfileProvider.future);
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _ProfileHeader(profile: profile),
            const SizedBox(height: 16),
            _SectionCard(
              title: 'Public Info',
              icon: Icons.public_outlined,
              subtitle: 'Visible to everyone',
              fieldConfigs: [
                _FieldConfig('Display Name', profile.publicInfo?.displayName),
                _FieldConfig('Bio', profile.publicInfo?.bio, multiline: true),
              ],
              onSave: (values) async {
                await ref.read(myProfileProvider.notifier).patchSection(
                      publicInfo: PublicInfo(
                        displayName: values[0],
                        bio: values[1],
                      ),
                    );
              },
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Friends Only',
              icon: Icons.people_outlined,
              subtitle: 'Visible to friends',
              fieldConfigs: [
                _FieldConfig('Phone', profile.friendsInfo?.phone),
                _FieldConfig('Location', profile.friendsInfo?.location),
              ],
              onSave: (values) async {
                await ref.read(myProfileProvider.notifier).patchSection(
                      friendsInfo: FriendsInfo(
                        phone: values[0],
                        location: values[1],
                      ),
                    );
              },
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Private',
              icon: Icons.lock_outlined,
              subtitle: 'Visible only to you',
              fieldConfigs: [
                _FieldConfig('Real Name', profile.privateInfo?.realName),
                _FieldConfig(
                  'Date of Birth',
                  profile.privateInfo?.dateOfBirth,
                  hint: 'e.g. 1990-06-15',
                ),
              ],
              onSave: (values) async {
                await ref.read(myProfileProvider.notifier).patchSection(
                      privateInfo: PrivateInfo(
                        realName: values[0],
                        dateOfBirth: values[1],
                      ),
                    );
              },
            ),
            const SizedBox(height: 12),
            _SectionCard(
              title: 'Music Preferences',
              icon: Icons.music_note_outlined,
              subtitle: 'Visible only to you',
              fieldConfigs: [
                _FieldConfig(
                  'Genres',
                  profile.musicPreferences?.genres,
                  hint: 'e.g. Rock, Jazz, Electronic',
                ),
                _FieldConfig(
                  'Favorite Artists',
                  profile.musicPreferences?.favoriteArtists,
                  hint: 'e.g. Daft Punk, Miles Davis',
                ),
              ],
              onSave: (values) async {
                await ref.read(myProfileProvider.notifier).patchSection(
                      musicPreferences: MusicPreferences(
                        genres: values[0],
                        favoriteArtists: values[1],
                      ),
                    );
              },
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }
}

class _ProfileHeader extends StatelessWidget {
  final UserProfile profile;
  const _ProfileHeader({required this.profile});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            CircleAvatar(
              radius: 28,
              backgroundColor: Theme.of(context).colorScheme.primaryContainer,
              child: Text(
                profile.displayName[0].toUpperCase(),
                style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                      color: Theme.of(context).colorScheme.onPrimaryContainer,
                    ),
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    profile.displayName,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  if (profile.email != null)
                    Text(
                      profile.email!,
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: Theme.of(context).colorScheme.outline,
                          ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _FieldConfig {
  final String label;
  final String? initialValue;
  final bool multiline;
  final String? hint;

  const _FieldConfig(
    this.label,
    this.initialValue, {
    this.multiline = false,
    this.hint,
  });
}

class _SectionCard extends StatefulWidget {
  final String title;
  final IconData icon;
  final String subtitle;
  final List<_FieldConfig> fieldConfigs;
  final Future<void> Function(List<String?>) onSave;

  const _SectionCard({
    required this.title,
    required this.icon,
    required this.subtitle,
    required this.fieldConfigs,
    required this.onSave,
  });

  @override
  State<_SectionCard> createState() => _SectionCardState();
}

class _SectionCardState extends State<_SectionCard> {
  bool _editing = false;
  bool _saving = false;
  late List<TextEditingController> _controllers;

  @override
  void initState() {
    super.initState();
    _controllers = _buildControllers();
  }

  @override
  void didUpdateWidget(covariant _SectionCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Sync controllers to refreshed profile data, but only when not editing so
    // the user's in-progress text is preserved during a background re-fetch.
    if (!_editing) {
      _disposeControllers();
      _controllers = _buildControllers();
    }
  }

  @override
  void dispose() {
    _disposeControllers();
    super.dispose();
  }

  List<TextEditingController> _buildControllers() => widget.fieldConfigs
      .map((f) => TextEditingController(text: f.initialValue ?? ''))
      .toList();

  void _disposeControllers() {
    for (final c in _controllers) {
      c.dispose();
    }
  }

  void _cancelEdit() {
    _disposeControllers();
    _controllers = _buildControllers();
    setState(() => _editing = false);
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final values = _controllers
          .map((c) => c.text.trim().isEmpty ? null : c.text.trim())
          .toList();
      await widget.onSave(values);
      if (mounted) {
        setState(() {
          _editing = false;
          _saving = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Save failed: $e'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(widget.icon,
                    size: 20, color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(widget.title,
                          style: Theme.of(context).textTheme.titleMedium),
                      Text(
                        widget.subtitle,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                              color: Theme.of(context).colorScheme.outline,
                            ),
                      ),
                    ],
                  ),
                ),
                if (!_editing)
                  IconButton(
                    icon: const Icon(Icons.edit_outlined),
                    tooltip: 'Edit',
                    onPressed: () => setState(() => _editing = true),
                  ),
              ],
            ),
            const Divider(height: 24),
            if (_editing) ...[
              ..._buildEditFields(),
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: _saving ? null : _cancelEdit,
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: _saving ? null : _save,
                    child: _saving
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('Save'),
                  ),
                ],
              ),
            ] else
              ..._buildReadFields(),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildReadFields() {
    final hasAny = widget.fieldConfigs
        .any((f) => f.initialValue != null && f.initialValue!.isNotEmpty);
    if (!hasAny) {
      return [
        Text(
          'No information added yet.',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.outline,
                fontStyle: FontStyle.italic,
              ),
        ),
      ];
    }
    return widget.fieldConfigs.map((f) {
      final hasValue = f.initialValue != null && f.initialValue!.isNotEmpty;
      return Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 130,
              child: Text(
                '${f.label}:',
                style: const TextStyle(fontWeight: FontWeight.w500),
              ),
            ),
            Expanded(
              child: Text(
                hasValue ? f.initialValue! : '—',
                style: hasValue
                    ? null
                    : TextStyle(
                        color: Theme.of(context).colorScheme.outline),
              ),
            ),
          ],
        ),
      );
    }).toList();
  }

  List<Widget> _buildEditFields() {
    return List.generate(widget.fieldConfigs.length, (i) {
      final f = widget.fieldConfigs[i];
      return Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: TextFormField(
          controller: _controllers[i],
          maxLines: f.multiline ? 3 : 1,
          decoration: InputDecoration(
            labelText: f.label,
            hintText: f.hint,
            border: const OutlineInputBorder(),
          ),
        ),
      );
    });
  }
}
