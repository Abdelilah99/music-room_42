import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/models/friendship.dart';
import '../../core/models/user.dart';
import 'friends_provider.dart';

class FriendsScreen extends ConsumerStatefulWidget {
  const FriendsScreen({super.key});

  @override
  ConsumerState<FriendsScreen> createState() => _FriendsScreenState();
}

class _FriendsScreenState extends ConsumerState<FriendsScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs;
  final _searchCtrl = TextEditingController();

  @override
  void initState() {
    super.initState();
    _tabs = TabController(length: 3, vsync: this);
    _searchCtrl.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _tabs.dispose();
    _searchCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(friendsProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Friends'),
        bottom: state.maybeWhen(
          data: (d) => TabBar(
            controller: _tabs,
            tabs: [
              Tab(text: 'Friends (${d.accepted.length})'),
              Tab(text: 'Incoming (${d.incoming.length})'),
              Tab(text: 'Sent (${d.outgoing.length})'),
            ],
          ),
          orElse: () => null,
        ),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _openAddFriend(context),
        tooltip: 'Add friend',
        child: const Icon(Icons.person_add_outlined),
      ),
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(onRetry: () => ref.invalidate(friendsProvider)),
        data: (d) => TabBarView(
          controller: _tabs,
          children: [
            _FriendsTab(
              friends: d.accepted,
              filter: _searchCtrl.text,
              searchCtrl: _searchCtrl,
              onUnfriend: (id) => ref.read(friendsProvider.notifier).unfriend(id),
              onTap: (userId) => context.push('/profile/users/$userId'),
            ),
            _RequestsTab(
              requests: d.incoming,
              onAccept: (id) => ref.read(friendsProvider.notifier).accept(id),
              onReject: (id) => ref.read(friendsProvider.notifier).reject(id),
            ),
            _OutgoingTab(
              requests: d.outgoing,
              onCancel: (id) => ref.read(friendsProvider.notifier).cancelRequest(id),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openAddFriend(BuildContext context) {
    return showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (_) => const _AddFriendSheet(),
    );
  }
}

// ─── Friends tab ────────────────────────────────────────────────────────────

class _FriendsTab extends ConsumerWidget {
  final List<Friend> friends;
  final String filter;
  final TextEditingController searchCtrl;
  final void Function(String friendshipId) onUnfriend;
  final void Function(String userId) onTap;

  const _FriendsTab({
    required this.friends,
    required this.filter,
    required this.searchCtrl,
    required this.onUnfriend,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final visible = filter.isEmpty
        ? friends
        : friends
            .where((f) => f.user.displayName
                .toLowerCase()
                .contains(filter.toLowerCase()))
            .toList();

    return RefreshIndicator(
      onRefresh: () => ref.refresh(friendsProvider.future),
      child: CustomScrollView(
        slivers: [
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
              child: SearchBar(
                controller: searchCtrl,
                hintText: 'Search friends…',
                leading: const Icon(Icons.search),
                trailing: [
                  if (filter.isNotEmpty)
                    IconButton(
                      icon: const Icon(Icons.clear),
                      onPressed: searchCtrl.clear,
                    ),
                ],
              ),
            ),
          ),
          if (visible.isEmpty)
            SliverFillRemaining(
              child: _EmptyState(
                icon: Icons.people_outline,
                message: filter.isEmpty
                    ? 'No friends yet — add one with the + button'
                    : 'No match for that name',
              ),
            )
          else
            SliverList.builder(
              itemCount: visible.length,
              itemBuilder: (_, i) => _FriendTile(
                friend: visible[i],
                onUnfriend: () => onUnfriend(visible[i].friendshipId),
                onTap: () => onTap(visible[i].user.id),
              ),
            ),
        ],
      ),
    );
  }
}

class _FriendTile extends StatelessWidget {
  final Friend friend;
  final VoidCallback onUnfriend;
  final VoidCallback onTap;

  const _FriendTile({
    required this.friend,
    required this.onUnfriend,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: _Avatar(friend.user.displayName),
      title: Text(friend.user.displayName),
      subtitle: Text(friend.user.email),
      onTap: onTap,
      trailing: IconButton(
        icon: const Icon(Icons.person_remove_outlined),
        tooltip: 'Unfriend',
        onPressed: () => _confirm(context),
      ),
    );
  }

  void _confirm(BuildContext context) {
    showDialog<void>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Unfriend'),
        content: Text('Remove ${friend.user.displayName}?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () {
              Navigator.pop(context);
              onUnfriend();
            },
            child: Text(
              'Unfriend',
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Incoming requests tab ───────────────────────────────────────────────────

class _RequestsTab extends ConsumerWidget {
  final List<FriendRequest> requests;
  final void Function(String id) onAccept;
  final void Function(String id) onReject;

  const _RequestsTab({
    required this.requests,
    required this.onAccept,
    required this.onReject,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return RefreshIndicator(
      onRefresh: () => ref.refresh(friendsProvider.future),
      child: requests.isEmpty
          ? const _EmptyScrollable(
              child: _EmptyState(
                icon: Icons.inbox_outlined,
                message: 'No incoming friend requests',
              ),
            )
          : ListView.builder(
              itemCount: requests.length,
              itemBuilder: (_, i) => _RequestTile(
                request: requests[i],
                onAccept: () => onAccept(requests[i].id),
                onReject: () => onReject(requests[i].id),
              ),
            ),
    );
  }
}

class _RequestTile extends StatelessWidget {
  final FriendRequest request;
  final VoidCallback onAccept;
  final VoidCallback onReject;

  const _RequestTile({
    required this.request,
    required this.onAccept,
    required this.onReject,
  });

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: _Avatar(request.user.displayName),
      title: Text(request.user.displayName),
      subtitle: Text(request.user.email),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.check_circle_outline),
            color: Colors.green,
            tooltip: 'Accept',
            onPressed: onAccept,
          ),
          IconButton(
            icon: const Icon(Icons.cancel_outlined),
            color: Colors.red,
            tooltip: 'Reject',
            onPressed: onReject,
          ),
        ],
      ),
    );
  }
}

// ─── Outgoing requests tab ───────────────────────────────────────────────────

class _OutgoingTab extends ConsumerWidget {
  final List<FriendRequest> requests;
  final void Function(String id) onCancel;

  const _OutgoingTab({required this.requests, required this.onCancel});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return RefreshIndicator(
      onRefresh: () => ref.refresh(friendsProvider.future),
      child: requests.isEmpty
          ? const _EmptyScrollable(
              child: _EmptyState(
                icon: Icons.send_outlined,
                message: 'No pending sent requests',
              ),
            )
          : ListView.builder(
              itemCount: requests.length,
              itemBuilder: (_, i) => ListTile(
                leading: _Avatar(requests[i].user.displayName),
                title: Text(requests[i].user.displayName),
                subtitle: Text(requests[i].user.email),
                trailing: TextButton(
                  onPressed: () => onCancel(requests[i].id),
                  child: const Text('Cancel'),
                ),
              ),
            ),
    );
  }
}

// ─── Add friend bottom sheet ─────────────────────────────────────────────────

class _AddFriendSheet extends ConsumerStatefulWidget {
  const _AddFriendSheet();

  @override
  ConsumerState<_AddFriendSheet> createState() => _AddFriendSheetState();
}

class _AddFriendSheetState extends ConsumerState<_AddFriendSheet> {
  final _ctrl = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final results = ref.watch(userSearchProvider(_query));

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: DraggableScrollableSheet(
        initialChildSize: 0.6,
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
              padding: const EdgeInsets.all(16),
              child: TextField(
                controller: _ctrl,
                autofocus: true,
                decoration: const InputDecoration(
                  hintText: 'Search by email or name…',
                  prefixIcon: Icon(Icons.search),
                  border: OutlineInputBorder(),
                ),
                onChanged: (v) => setState(() => _query = v.trim()),
              ),
            ),
            Expanded(
              child: results.when(
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (_, _) =>
                    const Center(child: Text('Search failed. Retry.')),
                data: (users) => users.isEmpty
                    ? _EmptyState(
                        icon: Icons.person_search_outlined,
                        message: _query.isEmpty
                            ? 'Type a name or email to search'
                            : 'No users found',
                      )
                    : ListView.builder(
                        controller: scrollCtrl,
                        itemCount: users.length,
                        itemBuilder: (_, i) => _SearchResultTile(user: users[i]),
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SearchResultTile extends ConsumerWidget {
  final User user;

  const _SearchResultTile({required this.user});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ListTile(
      leading: _Avatar(user.displayName),
      title: Text(user.displayName),
      subtitle: Text(user.email),
      trailing: FilledButton.tonal(
        onPressed: () => _send(context, ref),
        child: const Text('Add'),
      ),
    );
  }

  Future<void> _send(BuildContext context, WidgetRef ref) async {
    try {
      await ref.read(friendsProvider.notifier).sendRequest(user.id);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Request sent to ${user.displayName}')),
        );
        Navigator.pop(context);
      }
    } catch (_) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Failed to send request')),
        );
      }
    }
  }
}

// ─── Shared helpers ──────────────────────────────────────────────────────────

class _Avatar extends StatelessWidget {
  final String name;

  const _Avatar(this.name);

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      child: Text(
        name.isNotEmpty ? name[0].toUpperCase() : '?',
        style: const TextStyle(fontWeight: FontWeight.bold),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final IconData icon;
  final String message;

  const _EmptyState({required this.icon, required this.message});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 48, color: Theme.of(context).colorScheme.outline),
          const SizedBox(height: 12),
          Text(
            message,
            style: Theme.of(context)
                .textTheme
                .bodyMedium
                ?.copyWith(color: Theme.of(context).colorScheme.outline),
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }
}

// Wraps the empty state in a scrollable so RefreshIndicator can pull on it.
class _EmptyScrollable extends StatelessWidget {
  final Widget child;

  const _EmptyScrollable({required this.child});

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (_, constraints) => SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        child: SizedBox(height: constraints.maxHeight, child: child),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  final VoidCallback onRetry;

  const _ErrorView({required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.wifi_off_outlined, size: 48),
          const SizedBox(height: 12),
          const Text('Failed to load friends'),
          const SizedBox(height: 8),
          FilledButton.tonal(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}
