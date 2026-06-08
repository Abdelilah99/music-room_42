import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/core/models/event.dart';
import 'package:music_room/features/track_vote/events_provider.dart';
import 'package:music_room/shared/widgets/state_widgets.dart';

class EventListScreen extends ConsumerStatefulWidget {
  const EventListScreen({super.key});

  @override
  ConsumerState<EventListScreen> createState() => _EventListScreenState();
}

class _EventListScreenState extends ConsumerState<EventListScreen> {
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
      ref.read(eventsProvider.notifier).search(value.trim());
    });
  }

  @override
  Widget build(BuildContext context) {
    final eventsAsync = ref.watch(eventsProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Events')),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/vote/create'),
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
                hintText: 'Search events',
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
            child: eventsAsync.when(
              loading: () => const AppLoadingWidget(),
              error: (_, _) => AppErrorWidget(
                message: 'Failed to load events. Check your connection.',
                onRetry: () => ref.invalidate(eventsProvider),
              ),
              data: (events) => RefreshIndicator(
                onRefresh: () => ref.read(eventsProvider.notifier).refresh(),
                child: events.isEmpty
                    ? ListView(
                        // ListView keeps pull-to-refresh working on an empty list.
                        children: const [
                          SizedBox(height: 120),
                          AppEmptyStateWidget(
                            icon: Icons.event_busy_outlined,
                            message: 'No events found',
                          ),
                        ],
                      )
                    : ListView.separated(
                        itemCount: events.length,
                        separatorBuilder: (_, _) => const Divider(height: 1),
                        itemBuilder: (_, i) => _EventTile(event: events[i]),
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EventTile extends ConsumerWidget {
  const _EventTile({required this.event});

  final Event event;

  String _formatWindow() {
    final start = event.voteStart;
    final end = event.voteEnd;
    if (start == null || end == null) return 'No vote window set';
    String fmt(DateTime d) {
      final local = d.toLocal();
      final mm = local.month.toString().padLeft(2, '0');
      final dd = local.day.toString().padLeft(2, '0');
      final hh = local.hour.toString().padLeft(2, '0');
      final mi = local.minute.toString().padLeft(2, '0');
      return '$mm/$dd $hh:$mi';
    }

    // Times are shown in the device local zone; label it so users in other
    // zones know what the window refers to.
    final tz = start.toLocal().timeZoneName;
    return '${fmt(start)} - ${fmt(end)} $tz';
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ownerName = ref.watch(ownerNameProvider(event.ownerId));

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      title: Text(
        event.name,
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
          Text(_formatWindow(),
              style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
      trailing: _VisibilityBadge(isPublic: event.isPublic),
      onTap: () => context.push('/events/${event.id}'),
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
