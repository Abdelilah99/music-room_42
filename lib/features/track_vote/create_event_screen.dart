import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:music_room/core/api/events_api.dart';
import 'package:music_room/core/services/location_service.dart';
import 'package:music_room/features/track_vote/events_provider.dart';
import 'package:music_room/shared/widgets/snackbar_helper.dart';

class CreateEventScreen extends ConsumerStatefulWidget {
  const CreateEventScreen({super.key});

  @override
  ConsumerState<CreateEventScreen> createState() => _CreateEventScreenState();
}

class _CreateEventScreenState extends ConsumerState<CreateEventScreen> {
  final _formKey = GlobalKey<FormState>();
  final _nameCtrl = TextEditingController();
  final _latCtrl = TextEditingController();
  final _lngCtrl = TextEditingController();
  final _radiusCtrl = TextEditingController();

  static const _location = LocationService();

  String _visibility = 'public';
  int _license = 0;
  DateTime? _voteStart;
  DateTime? _voteEnd;
  bool _locating = false;
  bool _submitting = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _latCtrl.dispose();
    _lngCtrl.dispose();
    _radiusCtrl.dispose();
    super.dispose();
  }

  // License 2 needs a geofence + time window; the other tiers do not.
  bool get _needsGeofence => _license == 2;

  Future<void> _useCurrentLocation() async {
    setState(() => _locating = true);
    try {
      final pos = await _location.currentPosition();
      if (!mounted) return;
      if (pos == null) {
        AppSnackBar.show(
          context,
          message:
              'Location unavailable. Enter coordinates manually below.',
          type: SnackBarType.warning,
        );
        return;
      }
      setState(() {
        _latCtrl.text = pos.lat.toStringAsFixed(6);
        _lngCtrl.text = pos.lng.toStringAsFixed(6);
      });
    } finally {
      if (mounted) setState(() => _locating = false);
    }
  }

  Future<void> _pickDateTime({required bool isStart}) async {
    final now = DateTime.now();
    final base = (isStart ? _voteStart : _voteEnd) ?? now;
    final date = await showDatePicker(
      context: context,
      initialDate: base,
      firstDate: now.subtract(const Duration(days: 1)),
      lastDate: now.add(const Duration(days: 365)),
    );
    if (date == null || !mounted) return;
    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(base),
    );
    if (time == null || !mounted) return;
    final picked =
        DateTime(date.year, date.month, date.day, time.hour, time.minute);
    setState(() {
      if (isStart) {
        _voteStart = picked;
      } else {
        _voteEnd = picked;
      }
    });
  }

  String _fmt(DateTime d) {
    final mm = d.month.toString().padLeft(2, '0');
    final dd = d.day.toString().padLeft(2, '0');
    final hh = d.hour.toString().padLeft(2, '0');
    final mi = d.minute.toString().padLeft(2, '0');
    return '${d.year}-$mm-$dd $hh:$mi';
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    // Time window is mandatory only for license 2; the date pickers are not
    // covered by form field validators so they are checked here.
    if (_needsGeofence) {
      if (_voteStart == null || _voteEnd == null) {
        AppSnackBar.show(context,
            message: 'License 2 needs a vote start and end time',
            type: SnackBarType.error);
        return;
      }
    }
    if (_voteStart != null &&
        _voteEnd != null &&
        !_voteEnd!.isAfter(_voteStart!)) {
      AppSnackBar.show(context,
          message: 'Vote end must be after vote start',
          type: SnackBarType.error);
      return;
    }

    setState(() => _submitting = true);
    try {
      final event = await ref.read(eventsApiProvider).create(
            name: _nameCtrl.text.trim(),
            visibility: _visibility,
            license: _license,
            lat: _needsGeofence ? double.tryParse(_latCtrl.text.trim()) : null,
            lng: _needsGeofence ? double.tryParse(_lngCtrl.text.trim()) : null,
            radius: _needsGeofence
                ? double.tryParse(_radiusCtrl.text.trim())
                : null,
            voteStart: _voteStart,
            voteEnd: _voteEnd,
          );
      ref.invalidate(eventsProvider);
      if (mounted) context.go('/events/${event.id}');
    } on DioException catch (e) {
      if (!mounted) return;
      final msg = e.response?.data is Map
          ? (e.response?.data['error'] as String?)
          : null;
      AppSnackBar.show(context,
          message: msg ?? 'Could not create the event. Try again.',
          type: SnackBarType.error);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Create event')),
      body: AbsorbPointer(
        absorbing: _submitting,
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextFormField(
                controller: _nameCtrl,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(
                  labelText: 'Event name',
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
              const SizedBox(height: 20),
              const Text('License'),
              const SizedBox(height: 8),
              SegmentedButton<int>(
                segments: const [
                  ButtonSegment(value: 0, label: Text('Open')),
                  ButtonSegment(value: 1, label: Text('Invite')),
                  ButtonSegment(value: 2, label: Text('Geofenced')),
                ],
                selected: {_license},
                onSelectionChanged: (s) {
                  setState(() => _license = s.first);
                  if (_license == 2 && _latCtrl.text.isEmpty) {
                    _useCurrentLocation();
                  }
                },
              ),
              const SizedBox(height: 8),
              Text(
                _licenseHint(),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              const SizedBox(height: 20),
              _DateTimeField(
                label: 'Vote start',
                value: _voteStart == null ? null : _fmt(_voteStart!),
                onTap: () => _pickDateTime(isStart: true),
              ),
              const SizedBox(height: 12),
              _DateTimeField(
                label: 'Vote end',
                value: _voteEnd == null ? null : _fmt(_voteEnd!),
                onTap: () => _pickDateTime(isStart: false),
              ),
              if (_needsGeofence) ...[
                const SizedBox(height: 24),
                const Divider(),
                const SizedBox(height: 8),
                Row(
                  children: [
                    const Expanded(
                      child: Text('Geofence location',
                          style: TextStyle(fontWeight: FontWeight.w600)),
                    ),
                    TextButton.icon(
                      onPressed: _locating ? null : _useCurrentLocation,
                      icon: _locating
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child:
                                  CircularProgressIndicator(strokeWidth: 2))
                          : const Icon(Icons.my_location, size: 18),
                      label: const Text('Use current'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                        controller: _latCtrl,
                        keyboardType: const TextInputType.numberWithOptions(
                            signed: true, decimal: true),
                        decoration: const InputDecoration(
                          labelText: 'Latitude',
                          border: OutlineInputBorder(),
                        ),
                        validator: _needsGeofence ? _coordValidator : null,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: TextFormField(
                        controller: _lngCtrl,
                        keyboardType: const TextInputType.numberWithOptions(
                            signed: true, decimal: true),
                        decoration: const InputDecoration(
                          labelText: 'Longitude',
                          border: OutlineInputBorder(),
                        ),
                        validator: _needsGeofence ? _coordValidator : null,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _radiusCtrl,
                  keyboardType:
                      const TextInputType.numberWithOptions(decimal: true),
                  inputFormatters: [
                    FilteringTextInputFormatter.allow(RegExp(r'[0-9.]')),
                  ],
                  decoration: const InputDecoration(
                    labelText: 'Radius (km)',
                    border: OutlineInputBorder(),
                  ),
                  validator: (v) {
                    if (!_needsGeofence) return null;
                    final r = double.tryParse((v ?? '').trim());
                    if (r == null || r <= 0) return 'Enter a radius in km';
                    return null;
                  },
                ),
              ],
              const SizedBox(height: 28),
              FilledButton(
                onPressed: _submitting ? null : _submit,
                child: _submitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Create event'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  String? _coordValidator(String? v) {
    final d = double.tryParse((v ?? '').trim());
    if (d == null) return 'Required';
    return null;
  }

  String _licenseHint() {
    switch (_license) {
      case 1:
        return 'Only invited users can vote.';
      case 2:
        return 'Only users inside the radius during the vote window can vote.';
      default:
        return 'Anyone who can see the event can vote.';
    }
  }
}

class _DateTimeField extends StatelessWidget {
  const _DateTimeField({
    required this.label,
    required this.value,
    required this.onTap,
  });

  final String label;
  final String? value;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
          suffixIcon: const Icon(Icons.schedule),
        ),
        child: Text(value ?? 'Not set',
            style: value == null
                ? TextStyle(color: Theme.of(context).hintColor)
                : null),
      ),
    );
  }
}
