import 'dart:async';

import 'api_client.dart';

/// Application-wide state: auth session + cached group data fetched from the
/// Go backend. Screens read from here instead of the hardcoded mock file.
///
/// Every fetch fails silently and keeps whatever was cached last, so the UI
/// degrades gracefully when the API is offline.
class AppState {
  AppState._();

  static final AppState instance = AppState._();
  static AppState get I => instance;

  // ---------------------------------------------------------------------------
  // Auth
  // ---------------------------------------------------------------------------

  String? token;
  Map<String, dynamic>? user;

  bool get isLoggedIn => token != null;
  String get userName => (user?['name'] as String?) ?? 'Admin';
  String get userRole => (user?['role'] as String?) ?? 'group_admin';

  // ---------------------------------------------------------------------------
  // Cache
  // ---------------------------------------------------------------------------

  Map<String, dynamic>? group;
  List<Map<String, dynamic>> members = [];
  List<Map<String, dynamic>> meetings = [];
  List<Map<String, dynamic>> transactions = [];
  List<Map<String, dynamic>> loans = [];
  List<Map<String, dynamic>> fines = [];
  List<Map<String, dynamic>> announcements = [];
  List<Map<String, dynamic>> smsActivity = [];

  Map<String, dynamic>? _memberById(String? id) {
    if (id == null) return null;
    for (final m in members) {
      if (m['id'] == id) return m;
    }
    return null;
  }

  /// Looks up a member by id, falling back to `/group/members/:id` or `/group`.
  Future<Map<String, dynamic>?> memberById(String? id) async {
    final cached = _memberById(id);
    if (cached != null) return cached;
    if (members.isEmpty) {
      await fetchMembers();
      final again = _memberById(id);
      if (again != null) return again;
    }
    if (id == null) return null;
    try {
      return await api.get('/group/members/$id');
    } catch (_) {
      return null;
    }
  }

  // ---------------------------------------------------------------------------
  // Group
  // ---------------------------------------------------------------------------

  String get groupName => (group?['name'] as String?) ?? 'PesaBox';
  String get groupType => (group?['type'] as String?) ?? 'Vikoba';
  String get groupLocation => (group?['location'] as String?) ?? '';

  double get groupSavings => _num(group, 'savings');
  double get groupShares => _num(group, 'shares');
  double get groupSocialFund => _num(group, 'social');
  double get groupLoansOut => _num(group, 'loans');
  int get memberCount => int.tryParse(_str(group, 'members')) ?? 0;

  Future<Map<String, dynamic>?> fetchGroup({bool refresh = false}) async {
    if (group != null && !refresh) return group;
    try {
      group = await api.get('/group');
    } catch (_) {
      // keep existing cache
    }
    return group;
  }

  // ---------------------------------------------------------------------------
  // Lists
  // ---------------------------------------------------------------------------

  Future<List<Map<String, dynamic>>> fetchMembers(
      {bool refresh = false}) async {
    if (members.isNotEmpty && !refresh) return members;
    try {
      members = await _list('/group/members');
    } catch (_) {}
    return members;
  }

  Future<List<Map<String, dynamic>>> fetchMeetings(
      {bool refresh = false}) async {
    if (meetings.isNotEmpty && !refresh) return meetings;
    try {
      meetings = await _list('/meetings');
    } catch (_) {}
    return meetings;
  }

  Future<List<Map<String, dynamic>>> fetchTransactions(
      {bool refresh = false}) async {
    if (transactions.isNotEmpty && !refresh) return transactions;
    try {
      transactions = await _list('/transactions');
    } catch (_) {}
    return transactions;
  }

  Future<List<Map<String, dynamic>>> fetchLoans({bool refresh = false}) async {
    if (loans.isNotEmpty && !refresh) return loans;
    try {
      loans = await _list('/loans');
    } catch (_) {}
    return loans;
  }

  Future<List<Map<String, dynamic>>> fetchFines({bool refresh = false}) async {
    if (fines.isNotEmpty && !refresh) return fines;
    try {
      fines = await _list('/fines');
    } catch (_) {}
    return fines;
  }

  Future<List<Map<String, dynamic>>> fetchAnnouncements(
      {bool refresh = false}) async {
    if (announcements.isNotEmpty && !refresh) return announcements;
    try {
      announcements = await _list('/announcements');
    } catch (_) {}
    return announcements;
  }

  Future<List<Map<String, dynamic>>> fetchSmsActivity(
      {bool refresh = false}) async {
    if (smsActivity.isNotEmpty && !refresh) return smsActivity;
    try {
      smsActivity = await _list('/sms/activity');
    } catch (_) {}
    return smsActivity;
  }

  Future<List<Map<String, dynamic>>> _list(String path) async {
    final raw = await api.getList(path);
    return raw
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList(growable: false);
  }

  // ---------------------------------------------------------------------------
  // Derived helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic>? get nextMeeting {
    for (final m in meetings) {
      if (m['status'] == 'upcoming') return m;
    }
    return meetings.isEmpty ? null : meetings.first;
  }

  int get meetingsHeld =>
      meetings.where((m) => m['status'] == 'completed').length;

  static const List<String> _months = [
    '', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
    'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec',
  ];

  /// Renders an ISO date (`2026-09-15`) as `15 Sep 2026`.
  String shortDate(Object? value) {
    final s = value?.toString() ?? '';
    final parts = s.split('-');
    if (parts.length != 3) return s;
    final month = int.tryParse(parts[1]) ?? 0;
    final monthName =
        (month >= 1 && month <= 12) ? _months[month] : parts[1];
    return '${parts[2]} $monthName ${parts[0]}';
  }

  String meetingSubtitle(Map<String, dynamic> meeting) {
    final time = meeting['time'] as String? ?? '';
    final date = shortDate(meeting['date']);
    return [date, time].where((s) => s.isNotEmpty).join(' · ');
  }

  /// Converts `21 Sep 2026 · 10:24 AM` to a short label like `21 Sep 2026`.
  String txnDateLabel(Object? value) {
    final s = value?.toString() ?? '';
    final sep = s.indexOf('·');
    return (sep >= 0 ? s.substring(0, sep) : s).trim();
  }

  /// `TZS 1,300,000`.
  String money(num? value) {
    final v = (value ?? 0).round();
    final s = v.toString();
    final buf = StringBuffer();
    for (var i = 0; i < s.length; i++) {
      if (i > 0 && (s.length - i) % 3 == 0) buf.write(',');
      buf.write(s[i]);
    }
    return 'TZS $buf';
  }

  String amountLabel(Map<String, dynamic> txn) {
    final amount = (_num(txn, 'amount')).round();
    final s = amount.toString();
    final buf = StringBuffer();
    for (var i = 0; i < s.length; i++) {
      if (i > 0 && (s.length - i) % 3 == 0) buf.write(',');
      buf.write(s[i]);
    }
    final sign = (txn['direction'] as String?) == 'out' ? '-' : '+';
    return '$sign$buf';
  }

  String initials(String name) {
    final parts = name
        .split(' ')
        .where((p) => p.isNotEmpty)
        .toList(growable: false);
    if (parts.isEmpty) return '?';
    final first = parts.first[0];
    final last = parts.length > 1 ? parts.last[0] : '';
    return ('$first$last').toUpperCase();
  }

  bool isCredit(Map<String, dynamic> txn) =>
      (txn['direction'] as String?) == 'in';

  /// Human-ish label for a transaction type.
  String txnTypeLabel(String type) {
    switch (type) {
      case 'contribution':
        return 'Contribution';
      case 'share':
        return 'Share purchase';
      case 'loan_disbursement':
        return 'Loan disbursement';
      case 'loan_repayment':
        return 'Loan repayment';
      case 'fine':
        return 'Fine payment';
      case 'social_fund':
        return 'Social fund';
      case 'expense':
        return 'Group expense';
      case 'withdrawal':
        return 'Withdrawal';
      default:
        return type.isEmpty ? 'Transaction' : type;
    }
  }

  // ---------------------------------------------------------------------------
  // Parsing helpers
  // ---------------------------------------------------------------------------

  static double _num(Map<String, dynamic>? m, String key) {
    final v = m?[key];
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }

  static String _str(Map<String, dynamic>? m, String key) =>
      (m?[key] as String?) ?? '';
}