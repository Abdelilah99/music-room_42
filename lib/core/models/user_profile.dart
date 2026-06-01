class PublicInfo {
  final String? displayName;
  final String? bio;

  const PublicInfo({this.displayName, this.bio});

  factory PublicInfo.fromJson(Map<String, dynamic> json) => PublicInfo(
        displayName: json['display_name'] as String?,
        bio: json['bio'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'display_name': displayName,
        'bio': bio,
      };

  PublicInfo copyWith({String? displayName, String? bio}) => PublicInfo(
        displayName: displayName ?? this.displayName,
        bio: bio ?? this.bio,
      );
}

class FriendsInfo {
  final String? phone;
  final String? location;

  const FriendsInfo({this.phone, this.location});

  factory FriendsInfo.fromJson(Map<String, dynamic> json) => FriendsInfo(
        phone: json['phone'] as String?,
        location: json['location'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'phone': phone,
        'location': location,
      };

  FriendsInfo copyWith({String? phone, String? location}) => FriendsInfo(
        phone: phone ?? this.phone,
        location: location ?? this.location,
      );
}

class PrivateInfo {
  final String? realName;
  final String? dateOfBirth;

  const PrivateInfo({this.realName, this.dateOfBirth});

  factory PrivateInfo.fromJson(Map<String, dynamic> json) => PrivateInfo(
        realName: json['real_name'] as String?,
        dateOfBirth: json['date_of_birth'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'real_name': realName,
        'date_of_birth': dateOfBirth,
      };

  PrivateInfo copyWith({String? realName, String? dateOfBirth}) => PrivateInfo(
        realName: realName ?? this.realName,
        dateOfBirth: dateOfBirth ?? this.dateOfBirth,
      );
}

class MusicPreferences {
  final String? genres;
  final String? favoriteArtists;

  const MusicPreferences({this.genres, this.favoriteArtists});

  factory MusicPreferences.fromJson(Map<String, dynamic> json) =>
      MusicPreferences(
        genres: json['genres'] as String?,
        favoriteArtists: json['favorite_artists'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'genres': genres,
        'favorite_artists': favoriteArtists,
      };

  MusicPreferences copyWith({String? genres, String? favoriteArtists}) =>
      MusicPreferences(
        genres: genres ?? this.genres,
        favoriteArtists: favoriteArtists ?? this.favoriteArtists,
      );
}

class UserProfile {
  final String id;
  final String? email;
  final PublicInfo? publicInfo;
  final FriendsInfo? friendsInfo;
  final PrivateInfo? privateInfo;
  final MusicPreferences? musicPreferences;

  const UserProfile({
    required this.id,
    this.email,
    this.publicInfo,
    this.friendsInfo,
    this.privateInfo,
    this.musicPreferences,
  });

  String get displayName {
    final name = publicInfo?.displayName;
    if (name != null && name.isNotEmpty) return name;
    return email ?? id;
  }

  factory UserProfile.fromJson(Map<String, dynamic> json) {
    Map<String, dynamic>? section(String key) {
      final v = json[key];
      return v is Map ? Map<String, dynamic>.from(v) : null;
    }

    final pubMap = section('public_info');
    final friMap = section('friends_info');
    final priMap = section('private_info');
    final musMap = section('music_preferences');

    return UserProfile(
      id: json['id'] as String,
      email: json['email'] as String?,
      publicInfo: pubMap != null ? PublicInfo.fromJson(pubMap) : null,
      friendsInfo: friMap != null ? FriendsInfo.fromJson(friMap) : null,
      privateInfo: priMap != null ? PrivateInfo.fromJson(priMap) : null,
      musicPreferences: musMap != null ? MusicPreferences.fromJson(musMap) : null,
    );
  }

  UserProfile copyWith({
    PublicInfo? publicInfo,
    FriendsInfo? friendsInfo,
    PrivateInfo? privateInfo,
    MusicPreferences? musicPreferences,
  }) =>
      UserProfile(
        id: id,
        email: email,
        publicInfo: publicInfo ?? this.publicInfo,
        friendsInfo: friendsInfo ?? this.friendsInfo,
        privateInfo: privateInfo ?? this.privateInfo,
        musicPreferences: musicPreferences ?? this.musicPreferences,
      );
}
