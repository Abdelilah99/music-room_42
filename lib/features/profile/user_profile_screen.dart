import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:music_room/core/models/user_profile.dart';
import 'package:music_room/features/profile/profile_provider.dart';

class UserProfileScreen extends ConsumerWidget {
  final String userId;
  const UserProfileScreen({super.key, required this.userId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(userProfileProvider(userId));

    return Scaffold(
      appBar: AppBar(
        title: profileAsync.when(
          data: (p) => Text(p.displayName),
          loading: () => const Text('Profile'),
          error: (_, _) => const Text('Profile'),
        ),
      ),
      body: profileAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                'Failed to load profile',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () => ref.invalidate(userProfileProvider(userId)),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (profile) => RefreshIndicator(
          onRefresh: () async {
            ref.invalidate(userProfileProvider(userId));
            await ref.read(userProfileProvider(userId).future);
          },
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              _UserHeader(profile: profile),
              const SizedBox(height: 16),
              if (profile.publicInfo != null) ...[
                _ReadOnlySection(
                  title: 'Public Info',
                  icon: Icons.public_outlined,
                  fields: [
                    _Field('Display Name', profile.publicInfo?.displayName),
                    _Field('Bio', profile.publicInfo?.bio),
                  ],
                ),
                const SizedBox(height: 12),
              ],
              if (profile.friendsInfo != null) ...[
                _ReadOnlySection(
                  title: 'Friends Info',
                  icon: Icons.people_outlined,
                  fields: [
                    _Field('Phone', profile.friendsInfo?.phone),
                    _Field('Location', profile.friendsInfo?.location),
                  ],
                ),
                const SizedBox(height: 12),
              ],
              if (profile.publicInfo == null && profile.friendsInfo == null)
                Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Text(
                      'No profile information available.',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: Theme.of(context).colorScheme.outline,
                            fontStyle: FontStyle.italic,
                          ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Field {
  final String label;
  final String? value;
  const _Field(this.label, this.value);
}

class _UserHeader extends StatelessWidget {
  final UserProfile profile;
  const _UserHeader({required this.profile});

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
              child: Text(
                profile.displayName,
                style: Theme.of(context).textTheme.titleLarge,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ReadOnlySection extends StatelessWidget {
  final String title;
  final IconData icon;
  final List<_Field> fields;

  const _ReadOnlySection({
    required this.title,
    required this.icon,
    required this.fields,
  });

  @override
  Widget build(BuildContext context) {
    final visible = fields.where((f) => f.value?.isNotEmpty == true).toList();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon,
                    size: 20, color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Text(title, style: Theme.of(context).textTheme.titleMedium),
              ],
            ),
            const Divider(height: 24),
            if (visible.isEmpty)
              Text(
                'No information shared in this section.',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: Theme.of(context).colorScheme.outline,
                      fontStyle: FontStyle.italic,
                    ),
              )
            else
              ...visible.map(
                (f) => Padding(
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
                      Expanded(child: Text(f.value!)),
                    ],
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
