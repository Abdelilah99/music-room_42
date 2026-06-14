class User {
  final String id;
  final String email;
  final String? name;

  const User({required this.id, required this.email, this.name});

  String get displayName => (name != null && name!.isNotEmpty) ? name! : email;

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        email: json['email'] as String,
        name: json['name'] as String?,
      );
}
