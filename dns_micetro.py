"""DNS Authenticator for Micetro."""
import datetime
import logging
import requests
import zope.interface
from certbot import errors
from certbot import interfaces
from certbot.plugins import dns_common
import os

logger = logging.getLogger(__name__)
DEBUG = os.environ.get('CERTBOT_DNS_MICETRO_DEBUG', '0') == '1'
if DEBUG:
    logging.basicConfig(level=logging.DEBUG)
    console = logging.StreamHandler()
    console.setLevel(logging.DEBUG)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    console.setFormatter(formatter)
    logger.addHandler(console)

@zope.interface.implementer(interfaces.IAuthenticator)
@zope.interface.provider(interfaces.IPluginFactory)
class Authenticator(dns_common.DNSAuthenticator):
    """DNS Authenticator for Micetro

    This Authenticator uses the Micetro API to fulfill a dns-01 challenge.
    """

    description = ('Obtain certificates using a DNS TXT record '
                   '(if you are using BlueCat Micetro for DNS).')
    ttl = 120

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.credentials = None

    @classmethod
    def add_parser_arguments(cls, add):  # pylint: disable=arguments-differ
        super().add_parser_arguments(add)
        add('credentials', help='Micetro credentials INI file.')

    def more_info(self):
        return ('This plugin configures a DNS TXT record to respond to a '
                'dns-01 challenge using the Micetro API.')

    def _setup_credentials(self):
        self.credentials = self._configure_credentials(
            'credentials',
            'Micetro credentials INI file',
            {
                'username': 'Username for Micetro account',
                'password': 'Password for Micetro account',
                'url': 'Micetro API URL (e.g., https://ipam.example.com/api/v2)',
                'view': '(Optional) DNS view to use (e.g., external).'
            }
        )

    def _perform(self, domain, validation_name, validation):
        self._get_micetro_client().add_txt_record(domain, validation_name, validation, self.ttl)

    def _cleanup(self, domain, validation_name, validation):
        self._get_micetro_client().del_txt_record(domain, validation_name, validation)

    def _get_micetro_client(self):
        return _MicetroClient(
            self.credentials.conf('username'),
            self.credentials.conf('password'),
            self.credentials.conf('url'),
            self.credentials.conf('view')
        )


class ZoneInfo:
    """
    Represents a DNS zone in Micetro.
    """

    def __init__(self, name, ref, view_ref):
        self.name = name
        self.ref = ref
        self.view_ref = view_ref

    def __repr__(self):
        return f'ZoneInfo(name={self.name}, ref={self.ref}, view_ref={self.view_ref})'

    def view_ref_default(self):
        """
        Returns the name of the view associated with this zone.
        """
        return self.view_ref if self.view_ref else 'default'


class _MicetroClient:
    """
    Encapsulates all communication with the Micetro API.
    """

    def __init__(self, username, password, url, view=None):
        self.username = username
        self.password = password
        self.base_url = url.rstrip('/')
        self.view = view
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json'
        })
        self._authenticate()

    def _authenticate(self):
        """
        Authenticate with the Micetro API using username and password to get a Bearer token.

        :raises certbot.errors.PluginError: if authentication fails
        """
        try:
            logger.debug('Authenticating with Micetro API at %s/micetro/sessions', self.base_url)

            response = self.session.post(f'{self.base_url}/micetro/sessions', json={
                'loginName': self.username,
                'password': self.password
            })
            response.raise_for_status()

            session_data = response.json()
            result = session_data.get('result', {})
            if result:
                token = result.get('session')
            else:
                token = None
            if not token:
                raise errors.PluginError(
                    'Authentication failed: No token returned from Micetro API'
                )

            # Update session headers with the Bearer token
            self.session.headers.update({
                'Authorization': f'Bearer {token}'
            })

            logger.debug('Successfully authenticated with Micetro API')
        except requests.exceptions.RequestException as e:
            logger.error('Authentication request failed: %s', e)
            raise errors.PluginError(
                f'Authentication failed: Unable to connect to Micetro API: {e}'
            )
        except Exception as e:
            logger.error('Authentication failed: %s', e)
            raise errors.PluginError(f'Authentication failed: {e}')

    def add_txt_record(self, domain, record_name, record_content, record_ttl):
        """
        Add a TXT record using the supplied information.

        :param str domain: The domain to use to look up the Micetro zone (view: external).
        :param str record_name: The record name (typically beginning with '_acme-challenge.').
        :param str record_content: The record content (typically the challenge validation).
        :param int record_ttl: The record TTL (number of seconds that the record
            may be cached).
        :raises certbot.errors.PluginError: if an error occurs when communicating
            with the Micetro API
        """
        zone = self._find_zone(domain)
        try:
            if not record_name.endswith('.'):
                logger.debug(f"Record name '{record_name}' does not end with .; adding it.")
                record_name += '.'
            if not record_name.endswith(zone.name):
                logger.debug(f"Record name '{record_name}' does not end with zone name '{zone.name}'; adding it.")
                record_name += zone.name
            if f'{zone.name}{zone.name}' in record_name:
                record_name = record_name.replace(f'{zone.name}{zone.name}', zone.name)
                logger.debug(f"Record name '{record_name}' contains zone name '{zone.name+zone.name}' twice; replacing it.")
            logger.debug(f"Attempting to add record to zone '{zone.name}' ({zone.ref}): '{record_name}' '{record_content}'")
            body = {
                'dnsRecords': [
                    {
                        "name": record_name,
                        "type": "TXT",
                        "ttl": record_ttl,
                        "data": record_content,
                        "comment": "CertBot TXT record",
                        "enabled": True,
                        "aging": 0,
                        "dnsZoneRef": zone.ref,
                    }
                ],
                "saveComment": f"TXT record added by Certbot "
                               f"{datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S.%f%z')}",
                "dnsZoneRef": zone.ref,
                "forceOverrideOfNamingConflictCheck": True
            }
            response = self.session.post(f'{self.base_url}/dnsRecords', json=body)
            response.raise_for_status()

        except Exception as e:
            logger.error('Encountered exception adding TXT record: %s', e)
            raise errors.PluginError(
                f'Error communicating with the Micetro API: {e}'
            )

        record_id = self._find_txt_record_id(zone, record_name.rstrip('.'), record_content)
        logger.debug('Successfully added TXT record with record_id: %s', record_id)

    def del_txt_record(self, domain, record_name, record_content):
        """
        Delete a TXT record using the supplied information.

        Note that both the record's name and content are used to ensure that
        similar records created concurrently (e.g., due to concurrent invocations
        of this plugin) are not deleted.

        Failures are logged but not raised.

        :param str domain: The domain to use to look up the Micetro zone.
        :param str record_name: The record name (typically beginning with '_acme-challenge.').
        :param str record_content: The record content (typically the challenge validation).
        """
        try:
            zone = self._find_zone(domain)
        except errors.PluginError as e:
            logger.debug('Encountered error finding zone during deletion: %s', e)
            return

        if zone:
            record_id = self._find_txt_record_id(zone, record_name, record_content)
            if record_id:
                try:
                    logger.debug(
                        'Attempting to remove txt record from zone %s: %s: %s',
                        zone.ref, record_name, record_content
                    )

                    response = self.session.delete(f'{self.base_url}/dnsRecords/{record_id}')
                    response.raise_for_status()

                except Exception as e:  # pylint: disable=broad-exception-caught
                    logger.warning('Encountered Exception deleting TXT record: %s', e)
            else:
                logger.debug('TXT record not found; no cleanup needed.')
        else:
            logger.debug('Zone not found; no cleanup needed.')

    def _find_view_ref(self):
        """
        Find the view reference for the given view.

        :returns: The view reference, if found.
        :rtype: str
        :raises certbot.errors.PluginError: if no view reference is found.
        """
        view = self.view
        if not view:
            logger.debug('No view specified; using blank view.')
            return None
        try:
            logger.debug('Attempting to find view id for view %s', view)
            response = self.session.get(
                f'{self.base_url}/dnsViews',
                params={'filter': view}
            )
            response.raise_for_status()
            result = response.json().get('result', [])
            all_views = result.get('dnsViews', [])
            if all_views:
                matching_views = [x for x in all_views if x.get('name') == view]
                if matching_views:
                    view_ref = matching_views[0]['ref']
                    logger.debug('Found view reference of %s for view %s',
                                 view_ref, view)
                    return view_ref
        except Exception as e:
            logger.debug('Encountered exception finding view reference: %s', e)
            raise errors.PluginError(
                f'Error finding view reference for {view}: {e}'
            )
        return None

    def _find_zone(self, domain) -> ZoneInfo:
        """
        Find the zone_id for a given domain.

        :param str domain: The domain for which to find the zone_id.
        :returns: The zone_id, if found.
        :rtype: str
        :raises certbot.errors.PluginError: if no zone_id is found.
        """
        view_ref = self._find_view_ref()
        zone_name_guesses = dns_common.base_domain_name_guesses(domain)
        for zone_name in zone_name_guesses:
            try:
                params = {'filter': zone_name}
                if view_ref:
                    params['dnsViewRef'] = view_ref
                logger.debug('Attempting to find zone_id for %s using name %s', domain, zone_name)
                response = self.session.get(
                    f'{self.base_url}/dnsZones',
                    params=params
                )
                response.raise_for_status()
                result = response.json().get('result', [])
                all_zones = result.get('dnsZones', [])
                filter_zones = [x for x in all_zones if x.get('name') == zone_name + '.']
                if filter_zones:
                    zone = all_zones[0]
                    logger.debug(
                        'Found zone_id of %s for %s using name %s',
                        zone['ref'], domain, zone_name
                    )
                    return ZoneInfo(zone.get('name'), zone.get('ref'), zone.get('dnsViewRef'))
            except Exception as e:  # pylint: disable=broad-exception-caught
                logger.debug('Encountered exception finding zone_id: %s', e)
        logger.debug(
            'Unable to find zone_id for %s using zone names: %s',
            domain, zone_name_guesses
        )
        raise errors.PluginError(
            f'Unable to determine zone_id for {domain} using zone names: '
            f'{zone_name_guesses}. Please confirm that the domain name has been '
            f'entered correctly and is already associated with the supplied '
            f'Micetro account.'
        )

    def _find_txt_record_id(self, zone: ZoneInfo, record_name, record_content):
        """
        Find the record_id for a TXT record with the given name and content.

        :param str zone: The zone which contains the record.
        :param str record_name: The record name (typically beginning with '_acme-challenge.').
        :param str record_content: The record content (typically the challenge validation).
        :returns: The record_id, if found.
        :rtype: str
        """

        try:
            logger.debug(
                'Attempting to find record_id for record %s: %s in zone %s',
                record_name, record_content, zone.ref
            )
            if not record_name.endswith('.'):
                record_name += '.'
            search_record_name = record_name.lower().replace(zone.name, '').rstrip('.')
            response = self.session.get(f'{self.base_url}/dnsZones/{zone.ref}/dnsRecords', params={
                'filter': search_record_name
            })
            response.raise_for_status()

            result = response.json().get('result', [])
            if result:
                all_records = result.get('dnsRecords', [])
            else:
                all_records = []
            records = [
                record for record in all_records
                if (record.get('data') == record_content
                    and record.get('name') + '.' + zone.name == record_name
                    and record.get('type') == 'TXT')
            ]
        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.debug('Encountered Exception getting TXT record_id: %s', e)
            records = []

        if records:
            # Cleanup is returning the system to the state we found it. If, for
            # some reason, there are multiple matching records, we only delete
            # one because we only added one.
            return records[0]['ref']
        logger.debug('Unable to find TXT record.')
        return None
