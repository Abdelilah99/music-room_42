import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/user.dart';
import '../../features/profile/friends_provider.dart';

/// Shows a bottom sheet listing the current user's accepted friends.
/// Returns the selected [User], or null if the sheet is dismissed.
Future<User?> showFriendPicker(BuildContext context) {
  return showModalBottomSheet<User>(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _FriendPickerSheet(),
  );
}

class _FriendPickerSheet extends ConsumerStatefulWidget {
  const _FriendPickerSheet();

  @override
  ConsumerState<_FriendPickerSheet> createState() => _FriendPickerSheetState();
}

class _FriendPickerSheetState extends ConsumerState<_FriendPickerSheet> {
  final _searchCtrl = TextEditingController();

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(friendsProvider);
    final filter = _searchCtrl.text.toLowerCase();

    return DraggableScrollableSheet(
      initialChildSize: 0.55,
      maxChildSize: 0.9,
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
                    'Select a friend',
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
              controller: _searchCtrl,
              decoration: const InputDecoration(
                hintText: 'Filter…',
                prefixIcon: Icon(Icons.search),
                isDense: true,
                border: OutlineInputBorder(),
              ),
              onChanged: (_) => setState(() {}),
            ),
          ),
          Expanded(
            child: state.when(
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (_, _) => const Center(child: Text('Failed to load friends')),
              data: (d) {
                final visible = filter.isEmpty
                    ? d.accepted
                    : d.accepted
                        .where((f) => f.user.displayName
                            .toLowerCase()
                            .contains(filter))
                        .toList();

                if (visible.isEmpty) {
                  return Center(
                    child: Text(
                      d.accepted.isEmpty
                          ? 'You have no friends yet'
                          : 'No match',
                      style: TextStyle(
                          color: Theme.of(context).colorScheme.outline),
                    ),
                  );
                }

                return ListView.builder(
                  controller: scrollCtrl,
                  itemCount: visible.length,
                  itemBuilder: (_, i) {
                    final friend = visible[i];
                    return ListTile(
                      leading: CircleAvatar(
                        child: Text(
                          friend.user.displayName[0].toUpperCase(),
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                      ),
                      title: Text(friend.user.displayName),
                      subtitle: Text(friend.user.email),
                      onTap: () => Navigator.pop(context, friend.user),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
